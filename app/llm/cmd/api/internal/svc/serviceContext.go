// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"go-zero-voice-agent/app/llm/cmd/api/internal/config"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/chatmessageservice"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/chatsessionservice"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/llmchatservice"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/llmconfigservice"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config         config.Config
	LlmConfigRpc   llmconfigservice.LlmConfigService
	LlmChatRpc     llmchatservice.LlmChatService
	ChatSessionRpc chatsessionservice.ChatSessionService
	ChatMessageRpc chatmessageservice.ChatMessageService
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:         c,
		LlmConfigRpc:   llmconfigservice.NewLlmConfigService(zrpc.MustNewClient(c.LlmRpcConf)),
		LlmChatRpc:     llmchatservice.NewLlmChatService(zrpc.MustNewClient(c.LlmRpcConf)),
		ChatSessionRpc: chatsessionservice.NewChatSessionService(zrpc.MustNewClient(c.LlmRpcConf)),
		ChatMessageRpc: chatmessageservice.NewChatMessageService(zrpc.MustNewClient(c.LlmRpcConf)),
	}
}
