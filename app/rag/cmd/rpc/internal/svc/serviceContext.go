package svc

import (
	"fmt"

	"go-zero-voice-agent/app/rag/cmd/rpc/internal/config"
	"go-zero-voice-agent/pkg/minioutil"
)

type ServiceContext struct {
	Config      config.Config
	MinioClient *minioutil.MinioClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	minioClient, err := minioutil.NewMinioClient(minioutil.MinioConfig{
		Endpoint:  c.MinioConfig.Endpoint,
		AccessKey: c.MinioConfig.AccessKey,
		SecretKey: c.MinioConfig.SecretKey,
		UseSSL:    c.MinioConfig.UseSSL,
	})
	if err != nil {
		panic(fmt.Sprintf("init minio client failed: %v", err))
	}

	return &ServiceContext{
		Config:      c,
		MinioClient: minioClient,
	}
}
