package logic

import (
	"context"
	"encoding/json"

	"go-zero-voice-agent/app/llmservice/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llmservice/cmd/rpc/pb"
	chatconsts "go-zero-voice-agent/app/llmservice/pkg/consts"
	publicconsts "go-zero-voice-agent/pkg/consts"
	"go-zero-voice-agent/app/mqueue/cmd/job/jobtype"

	"github.com/hibiken/asynq"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateChatLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateChatLogic {
	return &CreateChatLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateChatLogic) CreateChat(in *pb.CreateChatReq) (*pb.CreateChatResp, error) {
	client := openai.NewClient(
		option.WithAPIKey(in.LlmConfig.ApiKey),
		option.WithBaseURL(in.LlmConfig.BaseUrl),
	)

	var chatMsgs []openai.ChatCompletionMessageParamUnion

	for i, msg := range in.Messages {
		switch msg.Role {
		case chatconsts.ChatMessageRoleUser:
			// 用户消息处理
			chatMsgs = append(chatMsgs, openai.UserMessage(msg.Content))
		case chatconsts.ChatMessageRoleAssistant:
			// 助手消息处理
			chatMsgs = append(chatMsgs, openai.AssistantMessage(msg.Content))
		case chatconsts.ChatMessageRoleSystem:
			// 系统消息处理
			if i != 0 {
				logx.Errorf("系统消息只能出现在第一条, 位置: %d", i)
			}
			chatMsgs = append(chatMsgs, openai.SystemMessage(msg.Content))
		default:
			logx.Errorf("未知的消息角色: %v", msg.Role)
		}
	}

	// 进行对话
	chatCompletion, err := client.Chat.Completions.New(
		context.TODO(), openai.ChatCompletionNewParams{
			Messages: openai.F(
				chatMsgs,
			),
			Model: openai.F(in.LlmConfig.Model),
		},
	)

	if err != nil {
		logx.Errorf("创建聊天失败, err: %v", err)
		return nil, err
	}

	
	aiRespMsg := &pb.ChatMsg{
		Role:    chatconsts.ChatMessageRoleAssistant,
		Content: chatCompletion.Choices[len(chatCompletion.Choices)-1].Message.Content,
	}

	go l.cacheConversation(chatCompletion.ID, in.Messages, aiRespMsg)

	return &pb.CreateChatResp{
		Id:      chatCompletion.ID,
		RespMsg: aiRespMsg,
	}, nil
}

func (l *CreateChatLogic) cacheConversation(conversationId string, userMsgs []*pb.ChatMsg, aiRespMsg *pb.ChatMsg) {
	defer func() {
		if r := recover(); r != nil {
			logx.Errorf("panic recovered in cacheConversation, err: %v", r)
		}
	}()

	cacheKey := publicconsts.ChatCacheKeyPrefix + conversationId
	if _, err := l.svcCtx.RedisClient.Del(cacheKey); err != nil {
		logx.Errorf("failed to clear conversation cache, key: %s, err: %v", cacheKey, err)
	}

	fullConversation := make([]*pb.ChatMsg, 0, len(userMsgs)+1)
	fullConversation = append(fullConversation, userMsgs...)
	fullConversation = append(fullConversation, aiRespMsg)
	for _, msg := range fullConversation {
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			logx.Errorf("failed to marshal message, err: %v", err)
			continue
		}

		if _, err = l.svcCtx.RedisClient.Rpush(cacheKey, string(msgBytes)); err != nil {
			logx.Errorf("fail to push message to Redis, key: %s, err: %v", cacheKey, err)
		}
	}

	l.svcCtx.RedisClient.Expire(cacheKey, chatconsts.ChatCacheExpireSeconds)

	task, err := jobtype.NewSyncChatMsgTask(conversationId)
	if err != nil {
		logx.Errorf("failed to create sync task for conversation %s, err: %v", conversationId, err)
		return
	}

	if _, err = l.svcCtx.AsynqClient.Enqueue(task, asynq.Queue(jobtype.QueueDefault)); err != nil {
		logx.Errorf("failed to enqueue sync task for conversation %s, err: %v", conversationId, err)
	}
}
