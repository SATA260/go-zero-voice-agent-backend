// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"go-zero-voice-agent/app/llm/cmd/rpc/client/chatmessageservice"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/chatsessionservice"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/llmchatservice"
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/config"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/client/asrconfigservice"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/client/ttsconfigservice"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config
	LlmChatServiceRpc llmchatservice.LlmChatService
	ChatSessionRpc    chatsessionservice.ChatSessionService
	ChatMessageRpc    chatmessageservice.ChatMessageService

	AsrConfigRpc asrconfigservice.AsrConfigService
	TtsConfigRpc ttsconfigservice.TtsConfigService
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		LlmChatServiceRpc: llmchatservice.NewLlmChatService(zrpc.MustNewClient(c.LlmRpcConf)),
		ChatSessionRpc: chatsessionservice.NewChatSessionService(zrpc.MustNewClient(c.LlmRpcConf)),
		ChatMessageRpc: chatmessageservice.NewChatMessageService(zrpc.MustNewClient(c.LlmRpcConf)),
		AsrConfigRpc: asrconfigservice.NewAsrConfigService(zrpc.MustNewClient(c.VoicechatRpcConf)),
		TtsConfigRpc: ttsconfigservice.NewTtsConfigService(zrpc.MustNewClient(c.VoicechatRpcConf)),
	}
}
