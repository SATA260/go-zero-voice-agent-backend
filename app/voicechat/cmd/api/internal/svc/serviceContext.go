// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"go-zero-voice-agent/app/llm/cmd/rpc/client/llmchatservice"
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/config"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config
	LlmChatServiceRpc llmchatservice.LlmChatService
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		LlmChatServiceRpc: llmchatservice.NewLlmChatService(zrpc.MustNewClient(c.LlmRpcConf)),
	}
}
