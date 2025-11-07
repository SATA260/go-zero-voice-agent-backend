// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"go-zero-voice-agent/app/llm/cmd/api/internal/config"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/llmconfigservice"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config       config.Config
	LlmConfigRpc llmconfigservice.LlmConfigService
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:       c,
		LlmConfigRpc: llmconfigservice.NewLlmConfigService(zrpc.MustNewClient(c.LlmRpcConf)),
	}
}
