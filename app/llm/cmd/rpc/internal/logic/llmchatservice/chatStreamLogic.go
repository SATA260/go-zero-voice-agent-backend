package llmchatservicelogic

import (
	"context"
	"errors"
	"io"
	"strings"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	chatconsts "go-zero-voice-agent/app/llm/pkg/consts"

	openai "github.com/sashabaranov/go-openai"
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
		return err
	}
	l.Logger.Infof("Collected %d history messages for conversation %s", len(historyMsgs), chatSession.ConvId)
	l.Logger.Debugf("History messages: %+v", historyMsgs)

	// 获取工具调用相关，并处理工具调用
	for _, msg := range in.Messages {
		if msg.GetRole() != chatconsts.ChatMessageRoleTool {
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
			if toolCall.Status == chatconsts.TOOL_CALLING_CONFIRMED {
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

			} else if toolCall.Status == chatconsts.TOOL_CALLING_REJECTED {
				l.Logger.Infof("User rejected tool execution: %s", toolCall.Info.Name)
				historyMsgs = append(historyMsgs, &pb.ChatMsg{
					Role:       chatconsts.ChatMessageRoleTool,
					Content:    "Tool execution was rejected by the user.",
					ToolCallId: msg.GetToolCallId(),
				})
			} else if toolCall.Status == chatconsts.TOOL_CALLING_FINISHED {
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
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// 递归处理多轮对话
	const maxRecursionDepth = 5
	for depth := 0; depth < maxRecursionDepth; depth++ {
		// 检查上下文是否取消
		if err := l.ctx.Err(); err != nil {
			return err
		}

		// 构建聊天请求
		chatCompletionReq := BuildChatCompletionRequest(in.LlmConfig, openaiMsgs, true, l.svcCtx.OpenaiToolList)
		l.Logger.Infof("OpenAI request (depth %d): %+v", depth, chatCompletionReq)

		chatStream, err := client.CreateChatCompletionStream(l.ctx, chatCompletionReq)
		if err != nil {
			return err
		}
		// 注意：在循环中 defer close 可能会导致资源泄露，应该显式 close
		// 但这里我们每次循环都会重新创建 stream，所以需要在循环结束前 close

		// 累积工具调用（因为流式返回会分片）
		type toolAcc struct {
			id        string
			name      string
			arguments strings.Builder
		}
		toolCallsAcc := map[string]*toolAcc{}
		// Map from index to tool call ID to handle chunks where ID is missing
		toolCallIndexMap := map[int]string{}

		var currentAssistantContent strings.Builder

		// 处理流式响应
	streamLoop:
		for {
			resp, err := chatStream.Recv()
			if errors.Is(err, io.EOF) {
				break streamLoop
			}

			// 处理接收错误
			if err != nil {
				chatStream.Close()
				stream.Send(&pb.ChatStreamResp{
					ConversationId: chatSession.ConvId,
					Error:          err.Error(),
				})
				return err
			}

			if len(resp.Choices) == 0 {
				continue
			}

			delta := resp.Choices[0].Delta

			// 处理文本内容
			if delta.Content != "" {
				currentAssistantContent.WriteString(delta.Content)
				stream.Send(&pb.ChatStreamResp{
					ConversationId: chatSession.ConvId,
					RespMsg: &pb.ChatMsg{
						Role:    chatconsts.ChatMessageRoleAssistant,
						Content: delta.Content,
					},
				})
			}

			// 检测并处理工具调用 (Tool Calling)
			if len(delta.ToolCalls) > 0 {
				for _, toolCall := range delta.ToolCalls {
					// 构建工具调用增量信息
					// 注意：在流式响应中，ID 和 Name 通常只在第一个包中出现
					// Arguments 会分散在后续的包中

					scope := chatconsts.TOOL_CALLING_SCOPE_SERVER
					requiresConfirmation := false
					status := ""

					// 尝试从 accumulated info 中获取 name，或者从当前 toolCall 中获取
					toolName := toolCall.Function.Name

					// 如果当前包没有 name，尝试查找已有的 accumulator
					if toolName == "" {
						if toolCall.ID != "" {
							if acc, ok := toolCallsAcc[toolCall.ID]; ok {
								toolName = acc.name
							}
						} else if toolCall.Index != nil {
							if id, ok := toolCallIndexMap[*toolCall.Index]; ok {
								if acc, ok := toolCallsAcc[id]; ok {
									toolName = acc.name
								}
							}
						}
					}

					if tool, ok := l.svcCtx.ToolRegistry[toolName]; ok {
						scope = tool.Scope()
						requiresConfirmation = tool.RequiresConfirmation()
					}

					if toolCall.ID != "" {
						status = chatconsts.TOOL_CALLING_START
						if requiresConfirmation {
							status = chatconsts.TOOL_CALLING_WAITING_CONFIRMATION
						}
					}

					// 聚合参数
					var currentAcc *toolAcc
					if toolCall.ID != "" {
						if toolCall.Index != nil {
							toolCallIndexMap[*toolCall.Index] = toolCall.ID
						}
						acc, ok := toolCallsAcc[toolCall.ID]
						if !ok {
							acc = &toolAcc{id: toolCall.ID, name: toolCall.Function.Name}
							toolCallsAcc[toolCall.ID] = acc
						}
						currentAcc = acc
					} else {
						if toolCall.Index != nil {
							if id, ok := toolCallIndexMap[*toolCall.Index]; ok {
								currentAcc = toolCallsAcc[id]
							}
						}
					}

					if currentAcc != nil {
						if toolCall.Function.Name != "" {
							currentAcc.name = toolCall.Function.Name
						}
						if toolCall.Function.Arguments != "" {
							currentAcc.arguments.WriteString(toolCall.Function.Arguments)
						}
					}

					resp := &pb.ChatStreamResp{
						ConversationId: chatSession.ConvId,
						RespMsg: &pb.ChatMsg{
							Role: chatconsts.ChatMessageRoleAssistant,
							ToolCalls: []*pb.ToolCall{
								{
									Info: &pb.ToolCallInfo{
										Id:                   toolCall.ID,
										Name:                 toolCall.Function.Name,
										ArgumentsJson:        toolCall.Function.Arguments,
										Scope:                scope,
										RequiresConfirmation: requiresConfirmation,
									},
									Status: status,
								},
							},
						},
					}
					stream.Send(resp)
				}
			}
		}
		chatStream.Close()

		// 本轮流结束

		// 构造完整的 Assistant Message
		assistantMsg := &pb.ChatMsg{
			Role:      chatconsts.ChatMessageRoleAssistant,
			Content:   currentAssistantContent.String(),
			ToolCalls: []*pb.ToolCall{},
		}

		// 转换 accumulated tools 到 OpenAI ToolCalls 和 PB ToolCalls
		// 需要保持顺序，这里简单遍历 map，如果顺序重要需要改进
		// OpenAI 的 ToolCalls 是 list，我们最好按 index 排序
		// 但 toolCallsAcc 是 map，我们用 toolCallIndexMap 来恢复顺序

		// Find max index
		maxIndex := -1
		for idx := range toolCallIndexMap {
			if idx > maxIndex {
				maxIndex = idx
			}
		}

		var orderedToolCalls []*toolAcc
		for i := 0; i <= maxIndex; i++ {
			if id, ok := toolCallIndexMap[i]; ok {
				if acc, ok := toolCallsAcc[id]; ok {
					orderedToolCalls = append(orderedToolCalls, acc)
				}
			}
		}

		openaiToolCalls := []openai.ToolCall{}

		for _, acc := range orderedToolCalls {
			tool, ok := l.svcCtx.ToolRegistry[acc.name]
			scope := chatconsts.TOOL_CALLING_SCOPE_SERVER
			requiresConfirmation := false
			if ok {
				scope = tool.Scope()
				requiresConfirmation = tool.RequiresConfirmation()
			}

			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, &pb.ToolCall{
				Info: &pb.ToolCallInfo{
					Id:                   acc.id,
					Name:                 acc.name,
					ArgumentsJson:        acc.arguments.String(),
					Scope:                scope,
					RequiresConfirmation: requiresConfirmation,
				},
			})

			openaiToolCalls = append(openaiToolCalls, openai.ToolCall{
				ID:   acc.id,
				Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{
					Name:      acc.name,
					Arguments: acc.arguments.String(),
				},
			})
		}

		// 缓存 Assistant 消息
		go l.svcCtx.CacheConversation(chatSession.ConvId, nil, assistantMsg)

		// 如果没有工具调用，说明对话结束
		if len(orderedToolCalls) == 0 {
			stream.Send(&pb.ChatStreamResp{
				ConversationId: chatSession.ConvId,
				IsComplete:     true,
			})
			return nil
		}

		// 有工具调用，添加到历史消息
		openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
			Role:      openai.ChatMessageRoleAssistant,
			Content:   currentAssistantContent.String(),
			ToolCalls: openaiToolCalls,
		})

		// 检查是否需要停止（等待客户端确认或执行）
		shouldStop := false

		// 检查是否有需要确认的工具调用
		for _, acc := range orderedToolCalls {
			tool, ok := l.svcCtx.ToolRegistry[acc.name]
			if !ok {
				l.Logger.Errorf("unknown tool called: %s", acc.name)
				continue
			}

			if tool.Scope() == chatconsts.TOOL_CALLING_SCOPE_CLIENT {
				shouldStop = true
			} else if tool.RequiresConfirmation() {
				shouldStop = true
			}
		}

		if shouldStop {
			// 如果需要停止，发送完成信号（客户端已经收到了工具调用流）
			stream.Send(&pb.ChatStreamResp{
				ConversationId: chatSession.ConvId,
				IsComplete:     true,
			})
			return nil
		}

		// 如果不需要停止（全是自动执行的工具），执行它们
		for _, acc := range orderedToolCalls {
			l.Logger.Infof("Auto-executing tool: %s", acc.name)
			content := ""

			tool, ok := l.svcCtx.ToolRegistry[acc.name]
			if ok {
				toolResult, err := tool.Execute(l.ctx, acc.arguments.String())
				if err != nil {
					content = "Error: " + err.Error()
				} else {
					content = toolResult
				}
			} else {
				content = "Error: Tool not found"
			}

			// 添加结果到 OpenAI 消息
			openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    content,
				ToolCallID: acc.id,
			})

			// 缓存工具结果
			toolMsg := &pb.ChatMsg{
				Role:       chatconsts.ChatMessageRoleTool,
				Content:    content,
				ToolCallId: acc.id,
			}
			go l.svcCtx.CacheConversation(chatSession.ConvId, nil, toolMsg)
		}

		// 继续下一轮循环（递归）
	}

	return status.Error(codes.Internal, "max recursion depth reached for tool calls")
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
