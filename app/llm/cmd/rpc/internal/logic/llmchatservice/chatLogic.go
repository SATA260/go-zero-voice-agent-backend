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
	l.Logger.Infof("History messages: %+v", historyMsgs)

	// 获取工具调用相关，并处理工具调用
	for _, msg := range in.Messages {
		if msg.GetRole() != consts.ChatMessageRoleTool {
			// 普通消息，直接加入历史
			historyMsgs = append(historyMsgs, msg)
			go l.svcCtx.CacheConversation(chatSession.ConvId, nil, msg)
			continue
		}

		if msg.GetToolCalls() == nil {
			l.Logger.Errorf("tool message missing tool_calls: %+v", msg)
			continue
		}

		for _, toolCall := range msg.ToolCalls {
			if toolCall.Status == consts.TOOL_CALLING_CONFIRMED {
				// 用户确认工具调用，执行它，并将结果加入历史消息
				tool, ok := l.svcCtx.ToolRegistry[toolCall.Info.Name]
				if !ok {
					l.Logger.Errorf("unknown tool called: %s", toolCall.Info.Name)
					continue
				}
				l.Logger.Infof("User confirmed tool execution: %s", toolCall.Info.Name)
				content := ""
				toolResult, err := tool.Execute(l.ctx, toolCall.Info.ArgumentsJson)
				if err != nil {
					content = "Error: " + err.Error()
				} else {
					content = toolResult
				}

				// 将执行结果作为 Tool 消息加入历史
				historyMsgs = append(historyMsgs, &pb.ChatMsg{
					Role:       chatconsts.ChatMessageRoleTool,
					Content:    content,
					ToolCallId: msg.GetToolCallId(),
				})

			} else if toolCall.Status == consts.TOOL_CALLING_REJECTED {
				l.Logger.Infof("User rejected tool execution: %s", toolCall.Info.Name)
				historyMsgs = append(historyMsgs, &pb.ChatMsg{
					Role:       chatconsts.ChatMessageRoleTool,
					Content:    "Tool execution was rejected by the user.",
					ToolCallId: msg.GetToolCallId(),
				})
			} else if toolCall.Status == consts.TOOL_CALLING_FINISHED {
				// 工具调用已完成，直接加入历史
				historyMsgs = append(historyMsgs, msg)
			}
		}

	}

	// 构建 OpenAI 格式的消息列表
	openaiMsgs := BuildOpenAIMessages(historyMsgs)

	// 创建 OpenAI 客户端
	client, err := NewOpenAIClient(in.LlmConfig)
	if err != nil {
		l.Logger.Errorf("NewOpenAIClient error: %v", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
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
	OpenaiToolListWithoutConfirm := l.svcCtx.OpenaiToolListWithoutConfirm

	// 构建并发送聊天完成请求
	req := BuildChatCompletionRequest(in.LlmConfig, openaiMsgs, false, OpenaiToolListWithoutConfirm)
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

	// 先缓存llm响应
	assistantMsg := &pb.ChatMsg{
		Role:    chatconsts.ChatMessageRoleAssistant,
		Content: choice.Message.Content,
		ToolCalls: []*pb.ToolCall{},
	}
	for _, toolCall := range choice.Message.ToolCalls {
		tool, ok := l.svcCtx.ToolRegistry[toolCall.Function.Name]
		if !ok {
			l.Logger.Errorf("unknown tool called: %s", toolCall.Function.Name)
			continue
		}
		toolCallMsg := &pb.ToolCall{
			Info: &pb.ToolCallInfo{
				Id:                   toolCall.ID,
				Name:                 toolCall.Function.Name,
				ArgumentsJson:        toolCall.Function.Arguments,
				Scope:                tool.Scope(), 
				RequiresConfirmation: tool.RequiresConfirmation(),                           
			},
		}
		assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, toolCallMsg)
	}
	go l.svcCtx.CacheConversation(chatSession.ConvId, nil, assistantMsg)

	// 没有工具调用，直接返回文本响应
	if len(choice.Message.ToolCalls) == 0 {
		respMsgs := []*pb.ChatMsg{assistantMsg}
		return &pb.ChatResp{
			ConversationId: chatSession.ConvId,
			RespMsg:        respMsgs,
		}, nil
	}


	// 有工具调用
	// 将响应消息放入历史消息
	openaiMsgs = append(openaiMsgs, choice.Message)

	// 用来存储返回给前端的需要确认的工具调用消息
	confirmMsg := &pb.ChatMsg{
		Role:      chatconsts.ChatMessageRoleAssistant,
		Content:   choice.Message.Content,
		ToolCalls: []*pb.ToolCall{},
	}

	// 先执行不需要确认的工具调用, 收集需要确认的工具调用
	for _, toolCall := range choice.Message.ToolCalls {
		tool, ok := l.svcCtx.ToolRegistry[toolCall.Function.Name]
		if !ok {
			l.Logger.Errorf("unknown tool called: %s", toolCall.Function.Name)
			continue
		}

		toolCallMsg := &pb.ToolCall{
			Info: &pb.ToolCallInfo{
				Id:                   toolCall.ID,
				Name:                 toolCall.Function.Name,
				ArgumentsJson:        toolCall.Function.Arguments,
				Scope:                tool.Scope(),
				RequiresConfirmation: tool.RequiresConfirmation(),
			},
			Status: consts.TOOL_CALLING_START,
		}

		// 在客户端执行的tool调用，直接放入confimMsg返回前端
		if tool.Scope() == consts.TOOL_CALLING_SCOPE_CLIENT {
			confirmMsg.ToolCalls = append(confirmMsg.ToolCalls, toolCallMsg)
			continue
		}

		// 服务端执行的调用
		// 需要确认的工具调用，放入confirmMsg返回前端
		if tool.RequiresConfirmation() {
			toolCallMsg.Status = consts.TOOL_CALLING_WAITING_CONFIRMATION
			confirmMsg.ToolCalls = append(confirmMsg.ToolCalls, toolCallMsg)
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
			Role:       chatconsts.ChatMessageRoleTool,
			Content:    content,
			ToolCallId: toolCall.ID,
		}
		go l.svcCtx.CacheConversation(chatSession.ConvId, nil, toolMsg)

	}

	// 如果有需要确认的工具调用，优先返回给前端
	if len(confirmMsg.ToolCalls) > 0 {
		return &pb.ChatResp{
			ConversationId: chatSession.ConvId,
			RespMsg:        []*pb.ChatMsg{confirmMsg},
		}, nil
	}

	// 没有需要确认的工具调用，继续递归处理
	return l.handleChatInteraction(in, chatSession, client, openaiMsgs, depth+1)
}
