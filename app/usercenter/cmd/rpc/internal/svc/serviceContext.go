package svc

import (
	"github.com/redis/go-redis/v9"
	"go-zero-voice-agent/app/usercenter/cmd/rpc/internal/config"
	"go-zero-voice-agent/app/usercenter/model"
)

type ServiceContext struct {
	Config      config.Config
	RedisClient *redis.Client

	UserModel     model.UserModel
	UserAuthModel model.UserAuthModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
	}
}
