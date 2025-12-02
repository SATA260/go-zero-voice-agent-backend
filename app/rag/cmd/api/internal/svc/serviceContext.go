// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"go-zero-voice-agent/app/rag/cmd/api/internal/config"
	"go-zero-voice-agent/app/rag/cmd/rpc/client/docservice"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config
	DocService docservice.DocService
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		DocService: docservice.NewDocService(zrpc.MustNewClient(c.RagRpcConf)),
	}
}
