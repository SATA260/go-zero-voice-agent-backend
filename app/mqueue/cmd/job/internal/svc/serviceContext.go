package svc

import (
	"go-zero-voice-agent/app/llm/model"
	"go-zero-voice-agent/app/mqueue/cmd/job/internal/config"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config           config.Config
	AsynqServer      *asynq.Server
	RedisClient   	 *redis.Redis
	ChatSessionModel model.ChatSessionModel
	ChatMessageModel model.ChatMessageModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewMysql(c.DB.DataSource)
	redisClient := redis.MustNewRedis(c.Redis)
	return &ServiceContext{
		Config:           c,
		AsynqServer:      newAsynqServer(c),
		RedisClient:      redisClient,
		ChatSessionModel: model.NewChatSessionModel(sqlConn, c.Cache),
		ChatMessageModel: model.NewChatMessageModel(sqlConn, c.Cache),
	}
}
