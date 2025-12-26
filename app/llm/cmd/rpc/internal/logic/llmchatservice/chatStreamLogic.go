package llmchatservicelogic

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"
	"go-zero-voice-agent/app/llm/pkg/consts"
	"go-zero-voice-agent/pkg/uniqueid"

	"github.com/sashabaranov/go-openai"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ChatStreamLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChatStreamLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatStreamLogic {
	return &ChatStreamLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ChatStream 处理流式聊天请求
// 1. 校验请求参数
// 2. 获取或创建会话
// 3. 收集历史消息
// 4. 处理输入消息中的工具调用结果（确认/拒绝/完成）
// 5. 构建 OpenAI 消息列表并初始化客户端
// 6. 调用 handleChatStreamInteraction 进行流式交互
func (l *ChatStreamLogic) ChatStream(in *pb.ChatStreamReq, stream pb.LlmChatService_ChatStreamServer) error {
	if err := l.validReq(in); err != nil {
		return err
	}

	// 获取或创建会话
	chatSession, err := GetOrCreateSession(l.ctx, l.svcCtx, in.ConversationId, in.UserId, in.Messages)
	if err != nil {
		return err
	}

	// 收集历史消息
	historyMsgs, err := CollectHistory(l.ctx, l.svcCtx, l.Logger, in.ConversationId, in.AutoFillHistory, in.LlmConfig, chatSession.Id)
	if err != nil {
		l.Logger.Errorf("collectHistory error: %v", err)
		return status.Error(codes.Internal, err.Error())
	}
	l.Logger.Infof("Collected %d history messages for conversation %s", len(historyMsgs), chatSession.ConvId)
	l.Logger.Debugf("History messages: %+v", historyMsgs)

	// 获取工具调用相关，并处理工具调用
	// 遍历输入消息，处理 Tool 类型的消息
	for _, msg := range in.Messages {
		if msg.GetRole() != consts.ChatMessageRoleTool {
			// 普通消息，直接加入历史并缓存
			historyMsgs = append(historyMsgs, msg)
			go l.svcCtx.CacheConversation(chatSession.ConvId, nil, msg)
			continue
		}

		if msg.GetToolCalls() == nil {
			l.Logger.Errorf("tool message missing tool_calls: %+v", msg)
			continue
		}

		// 标记需要落库的工具调用消息
		shouldCacheToolMsg := false
		var updatedAssistantMsg *pb.ChatMsg

		// 处理工具调用状态
		for _, toolCall := range msg.ToolCalls {
			// 找到最近的 assistant 消息并更新其中的 toolCalls
			if updatedAssistantMsg == nil {
				for i := len(historyMsgs) - 1; i >= 0; i-- {
					exist := historyMsgs[i]
					if exist.GetRole() != consts.ChatMessageRoleAssistant || len(exist.GetToolCalls()) == 0 {
						continue
					}
					for _, existingTc := range exist.ToolCalls {
						if existingTc.GetInfo() != nil && existingTc.GetInfo().GetId() == toolCall.GetInfo().GetId() {
							updatedAssistantMsg = exist
							break
						}
					}
					if updatedAssistantMsg != nil {
						break
					}
				}
			}
			if toolCall.Status == consts.TOOL_CALLING_CONFIRMED {
				// 用户确认工具调用，执行它，并将结果加入历史消息
				tool, ok := l.svcCtx.ToolRegistry[toolCall.Info.Name]
				if !ok {
					l.Logger.Errorf("unknown tool called: %s", toolCall.Info.Name)
					continue
				}
				l.Logger.Infof("User confirmed tool execution: %s", toolCall.Info.Name)
				content := ""
				toolCall.Status = consts.TOOL_CALLING_EXECUTING
				toolResult, err := tool.Execute(l.ctx, toolCall.Info.ArgumentsJson)
				if err != nil {
					content = "Error: " + err.Error()
					toolCall.Status = consts.TOOL_CALLING_FAILED
					toolCall.Error = err.Error()
				} else {
					content = toolResult
					toolCall.Status = consts.TOOL_CALLING_FINISHED
					toolCall.Result = toolResult
				}

				// 将执行结果更新回 assistant 消息
				if updatedAssistantMsg != nil {
					for _, existingTc := range updatedAssistantMsg.ToolCalls {
						if existingTc.GetInfo() != nil && existingTc.GetInfo().GetId() == toolCall.GetInfo().GetId() {
							existingTc.Status = toolCall.Status
							existingTc.Result = toolCall.Result
							existingTc.Error = toolCall.Error
						}
					}
				}

				l.Logger.Infof("Tool %s executed with result: %s", toolCall.Info.Name, content)
				shouldCacheToolMsg = true

			} else if toolCall.Status == consts.TOOL_CALLING_REJECTED {
				// 用户拒绝工具调用，记录拒绝信息
				l.Logger.Infof("User rejected tool execution: %s", toolCall.Info.Name)
				toolCall.Status = consts.TOOL_CALLING_REJECTED
				if updatedAssistantMsg != nil {
					for _, existingTc := range updatedAssistantMsg.ToolCalls {
						if existingTc.GetInfo() != nil && existingTc.GetInfo().GetId() == toolCall.GetInfo().GetId() {
							existingTc.Status = consts.TOOL_CALLING_REJECTED
							existingTc.Error = toolCall.Error
						}
					}
				}
				shouldCacheToolMsg = true
			} else if toolCall.Status == consts.TOOL_CALLING_FINISHED {
				// 工具调用已完成（可能是客户端执行的），更新 assistant 消息并标记缓存
				if updatedAssistantMsg != nil {
					for _, existingTc := range updatedAssistantMsg.ToolCalls {
						if existingTc.GetInfo() != nil && existingTc.GetInfo().GetId() == toolCall.GetInfo().GetId() {
							existingTc.Status = consts.TOOL_CALLING_FINISHED
							existingTc.Result = toolCall.Result
							existingTc.Error = toolCall.Error
						}
					}
				}
				shouldCacheToolMsg = true
			}
		}

		if shouldCacheToolMsg {
			if updatedAssistantMsg != nil {
				go l.svcCtx.UpdateAssistantToolCalls(chatSession.ConvId, updatedAssistantMsg)
			} else {
				go l.svcCtx.CacheConversation(chatSession.ConvId, nil, msg)
			}
		}
	}

	// 构建 OpenAI 格式的消息列表
	openaiMsgs := BuildOpenAIMessages(historyMsgs)

	// 创建 OpenAI 客户端
	client, err := NewOpenAIClient(in.LlmConfig)
	if err != nil {
		l.Logger.Errorf("NewOpenAIClient error: %v", err)
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// 开始流式交互处理
	return l.handleChatStreamInteraction(in, chatSession, client, openaiMsgs, 0, stream)
}

// handleChatStreamInteraction 处理流式聊天交互的核心逻辑
// 支持递归调用以处理多轮工具调用
// depth: 当前递归深度，防止无限循环
func (l *ChatStreamLogic) handleChatStreamInteraction(
	in *pb.ChatStreamReq,
	chatSession *model.ChatSession,
	client *openai.Client,
	openaiMsgs []openai.ChatCompletionMessage,
	depth int,
	stream pb.LlmChatService_ChatStreamServer,
) error {
	// 递归深度限制
	if depth >= maxRecursionDepth {
		return status.Error(codes.Internal, "max recursion depth reached for tool calls")
	}

	// 获取不需要确认即可执行的工具列表（用于 OpenAI 请求）
	// OpenaiToolListWithoutConfirm := l.svcCtx.OpenaiToolListWithoutConfirm
	OpenaiToolList := l.svcCtx.OpenaiToolList

	// 构建并发送聊天完成请求
	// stream=true 开启流式模式
	req := BuildChatCompletionRequest(in.LlmConfig, openaiMsgs, true, OpenaiToolList)
	l.Logger.Infof("OpenAI request (depth %d): %+v", depth, req)

	streamResp, err := client.CreateChatCompletionStream(l.ctx, req)
	if err != nil {
		l.Logger.Errorf("CreateChatCompletionStream error: %v", err)
		return status.Errorf(codes.Internal, "create chat completion stream failed: %v", err)
	}
	defer streamResp.Close()

	var fullContent strings.Builder
	// Map to store tool calls being built. Key is index.
	// 用于存储流式返回中构建的工具调用，Key 是索引
	toolCallsMap := make(map[int]*openai.ToolCall)

	// 循环读取流式响应
	for {
		response, err := streamResp.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			l.Logger.Errorf("Stream Recv error at depth %d: %v", depth, err)
			// 如果已经接收到部分内容，需要缓存
			if fullContent.Len() > 0 {
				l.Logger.Infof("Caching partial response due to stream error, depth %d, content length: %d",
					depth, fullContent.Len())
				assistantMsg := &pb.ChatMsg{
					Role:      consts.ChatMessageRoleAssistant,
					Content:   fullContent.String(),
					ToolCalls: []*pb.ToolCall{},
				}
				assistantMsg.MessageId = uniqueid.GenId()
				go l.svcCtx.CacheConversation(chatSession.ConvId, nil, assistantMsg)
			}
			return err
		}

		if len(response.Choices) == 0 {
			continue
		}

		delta := response.Choices[0].Delta

		// 处理文本内容
		if delta.Content != "" {
			fullContent.WriteString(delta.Content)
			// 实时发送分片给客户端
			if err := stream.Send(&pb.ChatStreamResp{
				ConversationId: chatSession.ConvId,
				RespMsg: &pb.ChatMsg{
					Role:    consts.ChatMessageRoleAssistant,
					Content: delta.Content,
				},
			}); err != nil {
				return err
			}
		}

		// 处理工具调用分片
		for _, toolCallDelta := range delta.ToolCalls {
			if toolCallDelta.Index == nil {
				continue
			}
			index := *toolCallDelta.Index
			if _, ok := toolCallsMap[index]; !ok {
				toolCallsMap[index] = &openai.ToolCall{
					Index: &index,
					Type:  toolCallDelta.Type,
					Function: openai.FunctionCall{
						Name:      toolCallDelta.Function.Name,
						Arguments: "",
					},
				}
			}

			// 拼接工具调用的各个部分
			if toolCallDelta.ID != "" {
				toolCallsMap[index].ID = toolCallDelta.ID
			}
			if toolCallDelta.Type != "" {
				toolCallsMap[index].Type = toolCallDelta.Type
			}
			if toolCallDelta.Function.Name != "" {
				toolCallsMap[index].Function.Name = toolCallDelta.Function.Name
			}
			if toolCallDelta.Function.Arguments != "" {
				toolCallsMap[index].Function.Arguments += toolCallDelta.Function.Arguments
			}
		}
	}

	// 将 map 转换为切片，并按索引排序（虽然 map 迭代顺序不确定，但这里我们只需要收集所有工具调用）
	var toolCalls []openai.ToolCall
	for i := 0; i < len(toolCallsMap); i++ {
		if tc, ok := toolCallsMap[i]; ok {
			toolCalls = append(toolCalls, *tc)
		}
	}

	l.Logger.Infof("Received response at depth %d: toolCalls count = %d, content length = %d",
		depth, len(toolCalls), len(fullContent.String()))

	// 构建完整的 Assistant 消息
	assistantMsg := &pb.ChatMsg{
		Role:      consts.ChatMessageRoleAssistant,
		Content:   fullContent.String(),
		ToolCalls: []*pb.ToolCall{},
	}
	assistantMsg.MessageId = uniqueid.GenId()

	// 如果没有工具调用，说明本次交互结束，只落一条消息
	if len(toolCalls) == 0 {
		l.Logger.Infof("Caching final assistant message for conversation %s, depth %d, content length: %d",
			chatSession.ConvId, depth, len(fullContent.String()))
		go l.svcCtx.CacheConversation(chatSession.ConvId, nil, assistantMsg)
		return nil
	}

	// 处理工具调用逻辑
	l.Logger.Infof("Processing %d tool calls at depth %d for conversation %s",
		len(toolCalls), depth, chatSession.ConvId)

	// 将 Assistant 消息加入历史，为下一轮（可能的）递归做准备
	openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
		Role:      openai.ChatMessageRoleAssistant,
		Content:   fullContent.String(),
		ToolCalls: toolCalls,
	})

	// 用于存储需要返回给前端确认的工具调用
	confirmMsg := &pb.ChatMsg{
		Role:      consts.ChatMessageRoleAssistant,
		Content:   fullContent.String(),
		ToolCalls: []*pb.ToolCall{},
	}
	confirmMsg.MessageId = assistantMsg.MessageId

	// 遍历所有工具调用，决定是自动执行还是请求确认
	for _, toolCall := range toolCalls {
		tool, ok := l.svcCtx.ToolRegistry[toolCall.Function.Name]
		if !ok {
			continue
		}

		toolCallMsg := &pb.ToolCall{
			Info: &pb.ToolCallInfo{
				Id:                   toolCall.ID,
				Name:                 toolCall.Function.Name,
				ArgumentsJson:        toolCall.Function.Arguments,
				Scope:                tool.Scope(),
				RequiresConfirmation: tool.RequiresConfirmation(),
				Description:          tool.Description(),
			},
			Status: consts.TOOL_CALLING_START,
		}

		// 记录到待持久化消息，避免重复落库
		assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, toolCallMsg)

		// 客户端执行的工具，直接加入确认消息列表返回给前端
		if tool.Scope() == consts.TOOL_CALLING_SCOPE_CLIENT {
			confirmMsg.ToolCalls = append(confirmMsg.ToolCalls, toolCallMsg)
			continue
		}

		// 需要确认的工具，标记状态并加入确认消息列表
		if tool.RequiresConfirmation() {
			toolCallMsg.Status = consts.TOOL_CALLING_WAITING_CONFIRMATION
			confirmMsg.ToolCalls = append(confirmMsg.ToolCalls, toolCallMsg)
			continue
		}

		// 自动执行工具
		// 特殊处理 RAG 工具：注入用户ID和文件ID
		if toolCall.Function.Name == consts.TOOL_CALLING_SELF_RAG {
			var argsMap map[string]interface{}
			err := json.Unmarshal([]byte(toolCall.Function.Arguments), &argsMap)
			if err != nil {
				l.Logger.Errorf("failed to unmarshal rag tool arguments: %v", err)
			} else {
				argsMap["user_id"] = in.UserId
				argsMap["file_ids"] = in.RagFileIds
				newArgs, err := json.Marshal(argsMap)
				if err != nil {
					l.Logger.Errorf("failed to marshal updated rag tool arguments: %v", err)
				} else {
					toolCall.Function.Arguments = string(newArgs)
				}
			}
		}

		l.Logger.Infof("Auto-executing tool: %s", toolCall.Function.Name)
		toolCallMsg.Status = consts.TOOL_CALLING_EXECUTING
		content := ""
		toolResult, err := tool.Execute(l.ctx, toolCall.Function.Arguments)
		if err != nil {
			content = "Error: " + err.Error()
			toolCallMsg.Status = consts.TOOL_CALLING_FAILED
			toolCallMsg.Error = err.Error()
		} else {
			content = toolResult
			toolCallMsg.Status = consts.TOOL_CALLING_FINISHED
			toolCallMsg.Result = toolResult
		}

		// 将工具执行结果加入 OpenAI 消息历史
		openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    content,
			ToolCallID: toolCall.ID,
		})

	}

	// 仅持久化一条包含工具调用状态/结果的消息
	go l.svcCtx.CacheConversation(chatSession.ConvId, nil, assistantMsg)

	// 如果有需要确认的工具调用，发送给客户端并结束本次流
	if len(confirmMsg.ToolCalls) > 0 {
		// Send confirmation request to client
		return stream.Send(&pb.ChatStreamResp{
			ConversationId: chatSession.ConvId,
			RespMsg:        confirmMsg,
		})
	}

	// 所有自动执行的工具都已执行完毕，递归调用以获取 LLM 对工具结果的响应
	return l.handleChatStreamInteraction(in, chatSession, client, openaiMsgs, depth+1, stream)
}

// 校验请求参数
func (l *ChatStreamLogic) validReq(in *pb.ChatStreamReq) error {
	if in == nil {
		return errors.New("empty request")
	}

	if in.LlmConfig == nil {
		return errors.New("missing llm config")
	}

	if in.LlmConfig.Model == "" {
		return errors.New("model is required")
	}

	if err := l.ctx.Err(); err != nil {
		return err
	}

	return nil
}
