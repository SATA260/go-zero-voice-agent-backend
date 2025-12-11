package llmchatservicelogic

import (
	"context"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/pkg/consts"
	chatconsts "go-zero-voice-agent/app/llm/pkg/consts"

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

	// 1. 校验请求参数
	if in == nil || in.LlmConfig == nil {
		return nil, status.Error(codes.InvalidArgument, "missing llm config")
	}
	if in.LlmConfig.Model == "" {
		return nil, status.Error(codes.InvalidArgument, "model is required")
	}

	if err := l.ctx.Err(); err != nil {
		return nil, err
	}

	// 2. 获取或创建会话
	chatSession, err := GetOrCreateSession(l.ctx, l.svcCtx, in.ConversationId, in.UserId, in.Messages)
	if err != nil {
		return nil, err
	}

	// 3. 收集历史消息
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

	// 4. 创建 OpenAI 客户端
	client, err := NewOpenAIClient(in.LlmConfig)
	if err != nil {
		l.Logger.Errorf("NewOpenAIClient error: %v", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// 5. 构建并发送聊天完成请求
	req := BuildChatCompletionRequest(in.LlmConfig, openaiMsgs, false)
	l.Logger.Infof("OpenAI request: %+v", req)

	completion, err := client.CreateChatCompletion(l.ctx, req)
	if err != nil {
		l.Logger.Errorf("CreateChatCompletion error: %v", err)
		return nil, status.Errorf(codes.Internal, "create chat completion failed: %v", err)
	}
	if len(completion.Choices) == 0 {
		return nil, status.Error(codes.Internal, "empty response from llm")
	}

	respMsgs := make([]*pb.ChatMsg, 0)

	choice := completion.Choices[0]
	if len(choice.Message.ToolCalls) > 0 {
		for _, toolCall := range choice.Message.ToolCalls {
			respMsgs = append(respMsgs, &pb.ChatMsg{
				Role: chatconsts.ChatMessageRoleTool,
				ToolCalls: &pb.ToolCallDelta{
					Id:             toolCall.ID,
					Name:           toolCall.Function.Name,
					ArgumentsJson:  toolCall.Function.Arguments,
					Status:         consts.TOOL_CALLING_START,
				},
			})
		}
	}

	// 6. 处理响应
	assistantMsg := &pb.ChatMsg{
		Role:    chatconsts.ChatMessageRoleAssistant,
		Content: choice.Message.Content,
	}
	l.Logger.Infof("LLM response content: %s", choice.Message.Content)

	// 异步缓存新消息（用户输入 + 助手响应）以避免重复
	go l.svcCtx.CacheConversation(chatSession.ConvId, in.Messages, assistantMsg)

	respMsgs = append(respMsgs, assistantMsg)

	return &pb.ChatResp{
		Id:      chatSession.ConvId,
		RespMsg: respMsgs,
	}, nil

}
