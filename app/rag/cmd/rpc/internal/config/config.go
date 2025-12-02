package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	DB struct {
		DataSource string
	}

	Cache cache.CacheConf

	MinioConfig struct {
		Endpoint  string
		AccessKey string
		SecretKey string
		UseSSL    bool
	}

	FastaptRagConfig struct {
		Endpoint string
	}
}
