package svc

import (
	"go-zero-voice-agent/app/voicechat/cmd/rpc/internal/config"
	"go-zero-voice-agent/app/voicechat/model"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config
	RedisClient *redis.Redis

	AsrConfigModel model.AsrConfigModel
	TtsConfigModel model.TtsConfigModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewMysql(c.DB.DataSource)

	return &ServiceContext{
		Config: c,
		RedisClient: redis.New(c.Redis.Host, func (r *redis.Redis)  {
			r.Type = c.Redis.Type
			r.Pass = c.Redis.Pass
		}),
		AsrConfigModel: model.NewAsrConfigModel(sqlConn, c.Cache),
		TtsConfigModel: model.NewTtsConfigModel(sqlConn, c.Cache),
	}
}
