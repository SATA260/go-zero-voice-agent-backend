package llmchatservicelogic

import (
	"context"
	"errors"
	"io"
	"strings"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/pkg/consts"

	openai "github.com/sashabaranov/go-openai"
	"github.com/zeromicro/go-zero/core/logx"
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

	// 将当前请求的消息追加到历史消息中
	if len(in.Messages) > 0 {
		historyMsgs = append(historyMsgs, in.Messages...)
	}
	openaiMsgs := BuildOpenAIMessages(historyMsgs)

	// 获取可用的工具列表
	openaiToolList := l.svcCtx.OpenaiToolList

	// 创建 OpenAI 客户端
	llmClient, err := NewOpenAIClient(in.LlmConfig)
	if err != nil {
		return err
	}

	// 构建聊天请求
	chatCompletionReq := BuildChatCompletionRequest(in.LlmConfig, openaiMsgs, true, openaiToolList)

	chatStream, err := llmClient.CreateChatCompletionStream(l.ctx, chatCompletionReq)
	if err != nil {
		return err
	}
	defer chatStream.Close()

	// 累积工具调用（因为流式返回会分片）
	type toolAcc struct {
		id        string
		name      string
		arguments strings.Builder
	}
	toolCallsAcc := map[string]*toolAcc{}

	// 处理流式响应
	for {
		resp, err := chatStream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		// 处理接收错误
		if err != nil {
			stream.Send(&pb.ChatStreamResp{
				Id: chatSession.ConvId,
				Payload: &pb.ChatStreamResp_Error{
					Error: err.Error(),
				},
			})
			break
		}

		if len(resp.Choices) == 0 {
			continue
		}

		delta := resp.Choices[0].Delta
		// 处理文本内容
		if delta.Content != "" {
			stream.Send(&pb.ChatStreamResp{
				Id: chatSession.ConvId,
				Payload: &pb.ChatStreamResp_Delta{
					Delta: &pb.StreamDelta{
						Content: delta.Content,
					},
				},
			})
		}

		// 检测并处理工具调用 (Tool Calling)
		if len(delta.ToolCalls) > 0 {
			for _, toolCall := range delta.ToolCalls {
				// 构建工具调用增量信息
				// 注意：在流式响应中，ID 和 Name 通常只在第一个包中出现
				// Arguments 会分散在后续的包中

				scope := consts.TOOL_CALLING_SCOPE_SERVER
				requiresConfirmation := false
				status := ""

				if tool, ok := l.svcCtx.ToolRegistry[toolCall.Function.Name]; ok {
					scope = consts.TOOL_CALLING_SCOPE_SERVER
					requiresConfirmation = tool.RequiresConfirmation()
				}

				if toolCall.ID != "" {
					status = consts.TOOL_CALLING_START
					if requiresConfirmation {
						status = consts.TOOL_CALLING_WAITING_CONFIRMATION
					}
				}

				// 聚合参数
				if toolCall.ID != "" {
					acc, ok := toolCallsAcc[toolCall.ID]
					if !ok {
						acc = &toolAcc{id: toolCall.ID, name: toolCall.Function.Name}
						toolCallsAcc[toolCall.ID] = acc
					}
					if toolCall.Function.Name != "" {
						acc.name = toolCall.Function.Name
					}
					if toolCall.Function.Arguments != "" {
						acc.arguments.WriteString(toolCall.Function.Arguments)
					}
				}

				stream.Send(&pb.ChatStreamResp{
					Id: chatSession.ConvId,
					Payload: &pb.ChatStreamResp_ToolCall{
						ToolCall: &pb.ToolCall{
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
				})
			}
		}

		// 检查是否因为工具调用而结束
		if resp.Choices[0].FinishReason == openai.FinishReasonToolCalls {
			// 汇总所有工具调用并发送最终状态（与 ChatLogic 对齐：区分客户端/服务端、是否需要确认）
			for _, acc := range toolCallsAcc {
				scope := consts.TOOL_CALLING_SCOPE_SERVER
				requiresConfirmation := false
				status := consts.TOOL_CALLING_FINISHED

				if tool, ok := l.svcCtx.ToolRegistry[acc.name]; ok {
					scope = tool.Scope()
					requiresConfirmation = tool.RequiresConfirmation()
					if scope == consts.TOOL_CALLING_SCOPE_CLIENT {
						// 客户端执行：交由前端处理
						status = consts.TOOL_CALLING_START
						if requiresConfirmation {
							status = consts.TOOL_CALLING_WAITING_CONFIRMATION
						}
					} else {
						// 服务端执行：如需确认则等待确认，否则标记已完成参数聚合
						if requiresConfirmation {
							status = consts.TOOL_CALLING_WAITING_CONFIRMATION
						}
					}
				}

				stream.Send(&pb.ChatStreamResp{
					Id: chatSession.ConvId,
					Payload: &pb.ChatStreamResp_ToolCall{
						ToolCall: &pb.ToolCall{
							Info: &pb.ToolCallInfo{
								Id:                   acc.id,
								Name:                 acc.name,
								ArgumentsJson:        acc.arguments.String(),
								Scope:                scope,
								RequiresConfirmation: requiresConfirmation,
							},
							Status: status,
						},
					},
				})
			}
		}
	}

	return nil
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
