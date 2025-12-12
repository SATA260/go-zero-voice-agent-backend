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

	// 0. 检查是否是针对某个 ToolCall 的确认请求
	// 约定：如果 Messages 中包含一条 Role=tool 且 Status 为 CONFIRMED/REJECTED 的消息，则视为确认指令
	if len(in.Messages) > 0 {
		lastMsg := in.Messages[len(in.Messages)-1]
		if lastMsg.Role == chatconsts.ChatMessageRoleTool && lastMsg.ToolCalls != nil {
			status := lastMsg.ToolCalls.Status
			if status == consts.TOOL_CALLING_CONFIRMED || status == consts.TOOL_CALLING_REJECTED {
				l.Logger.Infof("Received tool confirmation request: %s, status: %s", lastMsg.ToolCalls.Id, status)
				return l.handleToolConfirmation(in, chatSession, client, openaiMsgs, lastMsg.ToolCalls.Id, status == consts.TOOL_CALLING_CONFIRMED)
			}
		}
		go l.svcCtx.CacheConversation(chatSession.ConvId, in.Messages, nil)
	}

	return l.handleChatInteraction(in, chatSession, client, openaiMsgs, 0)
}

// handleToolConfirmation 处理工具确认逻辑
func (l *ChatLogic) handleToolConfirmation(
	in *pb.ChatReq,
	chatSession *model.ChatSession,
	client *openai.Client,
	openaiMsgs []openai.ChatCompletionMessage,
	toolCallID string,
	confirmed bool,
) (*pb.ChatResp, error) {
	// 1. 找到对应的 ToolCall (通常在历史记录的最后一条 Assistant 消息中)
	var toolCallToExecute *openai.ToolCall
	found := false

	// 从后往前找，找到最近的一条 Assistant 消息
	for i := len(openaiMsgs) - 1; i >= 0; i-- {
		msg := openaiMsgs[i]
		if msg.Role == openai.ChatMessageRoleAssistant && len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				if tc.ID == toolCallID {
					toolCallToExecute = &tc
					found = true
					break
				}
			}
		}
		if found {
			break
		}
	}

	if !found || toolCallToExecute == nil {
		return nil, status.Errorf(codes.NotFound, "tool call %s not found in history", toolCallID)
	}

	var content string
	if confirmed {
		// 2. 如果 confirmed == true，执行工具
		tool, ok := l.svcCtx.ToolRegistry[toolCallToExecute.Function.Name]
		if !ok {
			content = "Error: Tool not found"
		} else {
			toolResult, err := tool.Execute(l.ctx, toolCallToExecute.Function.Arguments)
			if err != nil {
				content = "Error: " + err.Error()
			} else {
				content = toolResult
			}
		}
	} else {
		// 3. 如果 confirmed == false，生成拒绝消息
		content = "User rejected the tool execution."
	}

	// 4. 将结果存入历史
	toolMsg := openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    content,
		ToolCallID: toolCallID,
	}
	openaiMsgs = append(openaiMsgs, toolMsg)

	// 异步存储 Tool 执行结果
	pbToolMsg := &pb.ChatMsg{
		Role:    chatconsts.ChatMessageRoleTool,
		Content: content,
		ToolCalls: &pb.ToolCallDelta{
			Id:     toolCallID,
			Status: consts.TOOL_CALLING_FINISHED,
			Result: content,
		},
	}
	if !confirmed {
		pbToolMsg.ToolCalls.Status = consts.TOOL_CALLING_REJECTED
	}
	go l.svcCtx.CacheConversation(chatSession.ConvId, nil, pbToolMsg)

	// 5. 递归调用 handleChatInteraction，继续后续对话
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

	// 判断是否有工具调用
	if len(choice.Message.ToolCalls) > 0 {
		// 用来存储返回给前端的需要确认的工具调用
		respMsgs = make([]*pb.ChatMsg, 0)

		// 先执行不需要确认的工具调用, 收集需要确认的工具调用
		for _, toolCall := range choice.Message.ToolCalls {
			tool, ok := l.svcCtx.ToolRegistry[toolCall.Function.Name]
			if !ok {
				l.Logger.Errorf("unknown tool called: %s", toolCall.Function.Name)
				continue
			}

			// 需要确认的工具调用，存储起来
			if tool.RequiresConfirmation() {
				respMsgs = append(respMsgs, &pb.ChatMsg{
					Role: chatconsts.ChatMessageRoleTool,
					ToolCalls: &pb.ToolCallDelta{
						Id:                   toolCall.ID,
						Name:                 toolCall.Function.Name,
						ArgumentsJson:        toolCall.Function.Arguments,
						Status:               consts.TOOL_CALLING_WAITING_CONFIRMATION,
						Scope:                tool.Scope(),
						RequiresConfirmation: tool.RequiresConfirmation(),
					},
				})
				continue
			}

			// 不需要确认的工具调用，执行它
			l.Logger.Infof("Auto-executing tool: %s", toolCall.Function.Name)
			content := ""
			toolResult, err := tool.Execute(l.ctx, toolCall.Function.Arguments)
			if err != nil {
				content = "Error: " + err.Error()
			} else {
				content = toolResult
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

			if len(respMsgs) > 0 {
				// 有需要确认的工具调用，先返回给前端
				return &pb.ChatResp{
					Id:      chatSession.ConvId,
					RespMsg: respMsgs,
				}, nil
			}

			// 递归调用自己，继续处理后续的对话
			return l.handleChatInteraction(in, chatSession, client, openaiMsgs, depth+1)
		}
	}

	// 没有工具调用，这时为最终文本回复
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
