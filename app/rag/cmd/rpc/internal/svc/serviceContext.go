package svc

import (
	"fmt"

	"go-zero-voice-agent/app/rag/cmd/rpc/internal/config"
	"go-zero-voice-agent/app/rag/model"
	"go-zero-voice-agent/pkg/minioutil"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config          config.Config
	MinioClient     *minioutil.MinioClient
	FileUploadModel model.FileUploadModel
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

	sqlConn := sqlx.NewMysql(c.DB.DataSource)

	return &ServiceContext{
		Config:          c,
		MinioClient:     minioClient,
		FileUploadModel: model.NewFileUploadModel(sqlConn, c.Cache),
	}
}
