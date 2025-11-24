package config

import (
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type Config struct {
	service.ServiceConf
	Redis redis.RedisConf
	Cache cache.CacheConf
	Asynq struct {
		Host string
		Pass string
		DB   int
	}
	DB struct {
		DataSource string
	}
}
