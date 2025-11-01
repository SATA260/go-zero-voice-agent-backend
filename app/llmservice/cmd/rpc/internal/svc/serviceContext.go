package svc

import (
	"encoding/json"
	"go-zero-voice-agent/app/llmservice/cmd/rpc/internal/config"
	"go-zero-voice-agent/app/llmservice/cmd/rpc/pb"
	"go-zero-voice-agent/app/llmservice/model"
	"go-zero-voice-agent/app/mqueue/cmd/job/jobtype"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	chatconsts "go-zero-voice-agent/app/llmservice/pkg/consts"
	publicconsts "go-zero-voice-agent/pkg/consts"
)

type ServiceContext struct {
	Config          config.Config
	ChatConfigModel model.ChatConfigModel
	ChatSessionModel model.ChatSessionModel
	ChatMessageModel model.ChatMessageModel

	RedisClient *redis.Redis
	AsynqClient *asynq.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewMysql(c.DB.DataSource)
	redisClient := redis.New(c.Redis.Host, func(r *redis.Redis) {
		r.Type = c.Redis.Type
		r.Pass = c.Redis.Pass
	})
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     c.Asynq.Host,
		Password: c.Asynq.Pass,
	})

	return &ServiceContext{
		Config:          c,
		RedisClient:     redisClient,
		AsynqClient:     asynqClient,
		ChatConfigModel: model.NewChatConfigModel(sqlConn, c.Cache),
		ChatSessionModel: model.NewChatSessionModel(sqlConn, c.Cache),
		ChatMessageModel: model.NewChatMessageModel(sqlConn, c.Cache),
	}
}

func (svc *ServiceContext) CacheConversation(conversationId string, userMsgs []*pb.ChatMsg, aiRespMsg *pb.ChatMsg) {
	defer func() {
		if r := recover(); r != nil {
			logx.Errorf("panic recovered in cacheConversation, err: %v", r)
		}
	}()

	cacheKey := publicconsts.ChatCacheKeyPrefix + conversationId
	if _, err := svc.RedisClient.Del(cacheKey); err != nil {
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

		if _, err = svc.RedisClient.Rpush(cacheKey, string(msgBytes)); err != nil {
			logx.Errorf("fail to push message to Redis, key: %s, err: %v", cacheKey, err)
		}
	}

	svc.RedisClient.Expire(cacheKey, chatconsts.ChatCacheExpireSeconds)

	task, err := jobtype.NewSyncChatMsgTask(conversationId)
	if err != nil {
		logx.Errorf("failed to create sync task for conversation %s, err: %v", conversationId, err)
		return
	}

	if _, err = svc.AsynqClient.Enqueue(task, asynq.Queue(jobtype.SyncChatMsgToDb)); err != nil {
		logx.Errorf("failed to enqueue sync task for conversation %s, err: %v", conversationId, err)
	}
}