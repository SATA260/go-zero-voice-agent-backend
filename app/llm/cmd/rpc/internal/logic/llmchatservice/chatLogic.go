package llmchatservicelogic

import (
	"context"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"
	"go-zero-voice-agent/app/llm/pkg/consts"
	chatconsts "go-zero-voice-agent/app/llm/pkg/consts"

	"github.com/sashabaranov/go-openai"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ChatLogic 处理聊天请求的逻辑结构体
type ChatLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewChatLogic 创建一个新的 ChatLogic 实例
func NewChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatLogic {
	return &ChatLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Chat 处理聊天请求，与 LLM 进行交互
func (l *ChatLogic) Chat(in *pb.ChatReq) (*pb.ChatResp, error) {
	l.Logger.Infof("Chat request: %+v", in)

	// 校验请求参数
	if in == nil || in.LlmConfig == nil {
		return nil, status.Error(codes.InvalidArgument, "missing llm config")
	}
	if in.LlmConfig.Model == "" {
		return nil, status.Error(codes.InvalidArgument, "model is required")
	}

	if err := l.ctx.Err(); err != nil {
		return nil, err
	}

	// 获取或创建会话
	chatSession, err := GetOrCreateSession(l.ctx, l.svcCtx, in.ConversationId, in.UserId, in.Messages)
	if err != nil {
		return nil, err
	}

	// 收集历史消息
	historyMsgs, err := CollectHistory(l.ctx, l.svcCtx, l.Logger, in.ConversationId, in.AutoFillHistory, in.LlmConfig, chatSession.Id)
	if err != nil {
		l.Logger.Errorf("collectHistory error: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	l.Logger.Infof("Collected %d history messages for conversation %s", len(historyMsgs), chatSession.ConvId)
	l.Logger.Debugf("History messages: %+v", historyMsgs)

	// 将当前请求的消息追加到历史消息中
	if len(in.Messages) > 0 {
		historyMsgs = append(historyMsgs, in.Messages...)
	}

	// 构建 OpenAI 格式的消息列表
	openaiMsgs := BuildOpenAIMessages(historyMsgs)

	// 创建 OpenAI 客户端
	client, err := NewOpenAIClient(in.LlmConfig)
	if err != nil {
		l.Logger.Errorf("NewOpenAIClient error: %v", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// 使用递归处理多轮工具调用
	// 首次调用时，先存储用户输入的消息
	if len(in.Messages) > 0 {
		go l.svcCtx.CacheConversation(chatSession.ConvId, in.Messages, nil)
	}
	return l.handleChatInteraction(in, chatSession, client, openaiMsgs, 0)
}

// handleChatInteraction 递归处理聊天交互，支持多轮工具调用
const maxRecursionDepth = 5

func (l *ChatLogic) handleChatInteraction(
	in *pb.ChatReq,
	chatSession *model.ChatSession,
	client *openai.Client,
	openaiMsgs []openai.ChatCompletionMessage,
	depth int,
) (*pb.ChatResp, error) {
	// 递归深度限制（原子计数器概念：depth 参数即为计数器）
	if depth >= maxRecursionDepth {
		return nil, status.Error(codes.Internal, "max recursion depth reached for tool calls")
	}

	// 获取可用的工具列表
	OpenaiToolList := l.svcCtx.OpenaiToolList

	// 1. 构建并发送聊天完成请求
	req := BuildChatCompletionRequest(in.LlmConfig, openaiMsgs, false, OpenaiToolList)
	l.Logger.Infof("OpenAI request (depth %d): %+v", depth, req)

	completion, err := client.CreateChatCompletion(l.ctx, req)
	if err != nil {
		l.Logger.Errorf("CreateChatCompletion error: %v", err)
		return nil, status.Errorf(codes.Internal, "create chat completion failed: %v", err)
	}
	if len(completion.Choices) == 0 {
		return nil, status.Error(codes.Internal, "empty response from llm")
	}

	choice := completion.Choices[0]
	respMsgs := make([]*pb.ChatMsg, 0)

	// 2. 判断是否有工具调用
	if len(choice.Message.ToolCalls) > 0 {
		// 2.1 预检查：是否有任何工具需要“前端确认”
		needsConfirmation := false
		for _, toolCall := range choice.Message.ToolCalls {
			tool, ok := l.svcCtx.ToolRegistry[toolCall.Function.Name]
			if ok && tool.RequiresConfirmation() {
				needsConfirmation = true
				break
			}
		}

		// 2.2 如果需要确认，直接中断递归，返回给前端
		if needsConfirmation {
			l.Logger.Info("Tool calls require confirmation, returning to client.")

			for _, toolCall := range choice.Message.ToolCalls {
				tool, ok := l.svcCtx.ToolRegistry[toolCall.Function.Name]
				scope := consts.TOOL_CALLING_SCOPE_SERVER
				requiresConfirm := false
				if ok {
					if tool.Scope() == consts.TOOL_CALLING_SCOPE_CLIENT {
						scope = consts.TOOL_CALLING_SCOPE_CLIENT
					}
					requiresConfirm = tool.RequiresConfirmation()
				}

				respMsgs = append(respMsgs, &pb.ChatMsg{
					Role: chatconsts.ChatMessageRoleTool,
					ToolCalls: &pb.ToolCallDelta{
						Id:                   toolCall.ID,
						Name:                 toolCall.Function.Name,
						ArgumentsJson:        toolCall.Function.Arguments,
						Status:               consts.TOOL_CALLING_WAITING_CONFIRMATION,
						Scope:                scope,
						RequiresConfirmation: requiresConfirm,
					},
				})
			}

			return &pb.ChatResp{
				Id:      chatSession.ConvId,
				RespMsg: respMsgs,
			}, nil
		}

		// 2.3 如果不需要确认，执行工具并递归调用
		l.Logger.Info("Auto-executing tools...")

		// 重要：必须将 Assistant 的 ToolCall 消息加入历史上下文
		openaiMsgs = append(openaiMsgs, choice.Message)

		for _, toolCall := range choice.Message.ToolCalls {
			tool, ok := l.svcCtx.ToolRegistry[toolCall.Function.Name]
			if !ok {
				l.Logger.Errorf("unknown tool called: %s", toolCall.Function.Name)
				openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    "Error: Tool not found",
					ToolCallID: toolCall.ID,
				})
				continue
			}

			// 执行工具
			toolResult, err := tool.Execute(l.ctx, toolCall.Function.Arguments)
			content := toolResult
			if err != nil {
				content = "Error: " + err.Error()
			}

			// 将执行结果作为 Tool 消息加入历史
			openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    content,
				ToolCallID: toolCall.ID,
			})

			// 存储 Tool 执行结果
			toolMsg := &pb.ChatMsg{
				Role:    chatconsts.ChatMessageRoleTool,
				Content: content,
				ToolCalls: &pb.ToolCallDelta{
					Id: toolCall.ID,
				},
			}
			go l.svcCtx.CacheConversation(chatSession.ConvId, nil, toolMsg)
		}

		// 递归调用下一轮
		return l.handleChatInteraction(in, chatSession, client, openaiMsgs, depth+1)

	}

	// 3. 没有工具调用，这是最终文本回复
	// 只有当 Content 不为空时，才构建文本消息
	if choice.Message.Content != "" {
		assistantMsg := &pb.ChatMsg{
			Role:    chatconsts.ChatMessageRoleAssistant,
			Content: choice.Message.Content,
		}
		l.Logger.Infof("LLM response content: %s", choice.Message.Content)

		// 异步缓存新消息（仅助手响应，用户输入已在首次调用时存储）
		go l.svcCtx.CacheConversation(chatSession.ConvId, nil, assistantMsg)

		respMsgs = append(respMsgs, assistantMsg)
	}

	return &pb.ChatResp{
		Id:      chatSession.ConvId,
		RespMsg: respMsgs,
	}, nil
}
