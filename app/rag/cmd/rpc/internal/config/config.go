package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf

	MinioConfig struct {
		Endpoint  string
		AccessKey string
		SecretKey string
		UseSSL    bool
	}
}
