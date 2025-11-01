package logic

import (
	"context"
	"encoding/json"

	"go-zero-voice-agent/app/llmservice/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llmservice/cmd/rpc/pb"

	chatconsts "go-zero-voice-agent/app/llmservice/pkg/consts"
	publicconsts "go-zero-voice-agent/pkg/consts"
	"go-zero-voice-agent/pkg/tool"

	"github.com/Masterminds/squirrel"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type ContinueChatLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewContinueChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContinueChatLogic {
	return &ContinueChatLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ContinueChatLogic) ContinueChat(in *pb.ContinueChatReq) (*pb.ContinueChatResp, error) {
	// 先从redis缓存中获取历史消息
	messages := make([]*pb.ChatMsg, 0, in.LlmConfig.ContentLength)

	cacheKey := publicconsts.ChatCacheKeyPrefix + in.Id
    msgStrs, err := l.svcCtx.RedisClient.Lrange(cacheKey, -int(in.LlmConfig.ContentLength), -1)
    if err != nil {
        // 如果从缓存获取失败，则从数据库获取
		session, err := l.svcCtx.ChatSessionModel.FindOneByConvId(l.ctx, in.Id)
		if err != nil {
			return nil, errors.Wrapf(err, "FindOneByConvId failed, convId: %s", in.Id)
		}

		queryBuilder := l.svcCtx.ChatMessageModel.SelectBuilder().Where(squirrel.Eq{"session_id": session.Id})
		pageMsgs, err := l.svcCtx.ChatMessageModel.FindPageListByPage(l.ctx, queryBuilder, 1, in.LlmConfig.ContentLength, "id DESC")
		if err != nil {
			return nil, errors.Wrapf(err, "FindPageListByPage failed, sessionId: %d", session.Id)
		}
		for _, msg := range pageMsgs {
			messages = append(messages, &pb.ChatMsg{
				Role:    msg.Role,
				Content: tool.NullStringToString(msg.Content),
			})
		}
    } else {
		for idx, raw := range msgStrs {
			var msg pb.ChatMsg
			if err := json.Unmarshal([]byte(raw), &msg); err != nil {
				return nil, errors.Wrapf(err, "decode cached message failed, key: %s, index: %d", cacheKey, idx)
			}
			messages = append(messages, &msg)
		}
	}
	
	var chatMsgs []openai.ChatCompletionMessageParamUnion

	for i, msg := range messages {
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
	client := openai.NewClient(
		option.WithAPIKey(in.LlmConfig.ApiKey),
		option.WithBaseURL(in.LlmConfig.BaseUrl),
	)
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

	go l.svcCtx.CacheConversation(in.Id, messages, aiRespMsg)

	return &pb.ContinueChatResp{
		Id:      in.Id,
		RespMsg: aiRespMsg,
	}, nil
}