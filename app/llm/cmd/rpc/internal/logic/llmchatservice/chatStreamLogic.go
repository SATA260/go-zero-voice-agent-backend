package llmchatservicelogic

import (
	"context"
	"errors"
	"io"

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

	// 创建 OpenAI 客户端
	llmClient, err := NewOpenAIClient(in.LlmConfig)
	if err != nil {
		return err
	}

	// 构建聊天请求
	chatCompletionReq := BuildChatCompletionRequest(in.LlmConfig, openaiMsgs, true)

	chatStream, err := llmClient.CreateChatCompletionStream(l.ctx, chatCompletionReq)
	if err != nil {
		return err
	}
	defer chatStream.Close()

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

		if len(resp.Choices) > 0 {
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

					var status string
					if toolCall.ID != "" {
						status = consts.TOOL_CALLING_START
					}

					stream.Send(&pb.ChatStreamResp{
						Id: chatSession.ConvId,
						Payload: &pb.ChatStreamResp_ToolCall{
							ToolCall: &pb.ToolCallDelta{
								Id:            toolCall.ID,
								Name:          toolCall.Function.Name,
								ArgumentsJson: toolCall.Function.Arguments,
								Status:        status,
							},
						},
					})
				}
			}

			// 检查是否因为工具调用而结束
			if resp.Choices[0].FinishReason == openai.FinishReasonToolCalls {
				// 发送一个 finished 状态，告知客户端工具调用参数接收完毕
				stream.Send(&pb.ChatStreamResp{
					Id: chatSession.ConvId,
					Payload: &pb.ChatStreamResp_ToolCall{
						ToolCall: &pb.ToolCallDelta{
							Status: consts.TOOL_CALLING_FINISHED,
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
