package svc

import (
	"go-zero-voice-agent/app/llmservice/model"
	"go-zero-voice-agent/app/mqueue/cmd/job/internal/config"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config           config.Config
	AsynqServer      *asynq.Server
	ChatCacheRedis   *redis.Redis
	ChatSessionModel model.ChatSessionModel
	ChatMessageModel model.ChatMessageModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewMysql(c.DB.DataSource)
	chatCacheRedis := redis.MustNewRedis(c.ChatCache)
	return &ServiceContext{
		Config:           c,
		AsynqServer:      newAsynqServer(c),
		ChatCacheRedis:   chatCacheRedis,
		ChatSessionModel: model.NewChatSessionModel(sqlConn, cache.CacheConf{}),
		ChatMessageModel: model.NewChatMessageModel(sqlConn, cache.CacheConf{}),
	}
}
