package svc

import (
	"go-zero-voice-agent/app/llmservice/cmd/rpc/internal/config"
	"go-zero-voice-agent/app/llmservice/model"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config          config.Config
	ChatConfigModel model.ChatConfigModel

	RedisClient *redis.Redis
	AsynqClient *asynq.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewMysql(c.DB.DataSource)
	redisClient := redis.MustNewRedis(c.Redis)
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     c.Asynq.Host,
		Password: c.Asynq.Pass,
	})

	return &ServiceContext{
		Config:          c,
		RedisClient:     redisClient,
		AsynqClient:     asynqClient,
		ChatConfigModel: model.NewChatConfigModel(sqlConn, c.Cache),
	}
}
