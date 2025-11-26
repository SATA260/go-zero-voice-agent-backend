// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package chat

import (
	"context"
	"strings"

	"go-zero-voice-agent/app/llm/cmd/api/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/api/internal/types"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/llmchatservice"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/llmconfigservice"
	chatconsts "go-zero-voice-agent/app/llm/pkg/consts"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type TextChatLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 进行文字对话
func NewTextChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TextChatLogic {
	return &TextChatLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TextChatLogic) TextChat(req *types.TextChatReq) (resp *types.TextChatResp, err error) {
	if req == nil {
		return nil, errors.New("invalid request")
	}

	trimmedMsg := strings.TrimSpace(req.Message)
	if trimmedMsg == "" {
		return nil, errors.New("message cannot be empty")
	}

	if req.ConfigId <= 0 {
		return nil, errors.New("configId must be greater than 0")
	}

	cfgResp, err := l.svcCtx.LlmConfigRpc.GetConfig(l.ctx, &llmconfigservice.GetConfigReq{Id: req.ConfigId})
	if err != nil {
		l.Logger.Errorf("LlmConfigRpc.GetConfig error, configId=%d, err=%v", req.ConfigId, err)
		return nil, errors.Wrap(err, "failed to fetch config")
	}

	cfg := cfgResp.GetConfig()
	if cfg == nil {
		return nil, errors.New("config not found")
	}

	if cfg.UserId != req.UserId {
		return nil, errors.New("not authorized to use this config")
	}

	llmCfg := buildChatLlmConfig(cfg)
	if llmCfg == nil {
		return nil, errors.New("invalid llm config")
	}

	if llmCfg.ApiKey == "" || llmCfg.Model == "" {
		return nil, errors.New("llm config missing api key or model")
	}

	convID := strings.TrimSpace(req.ConversationId)
	systemPrompt := strings.TrimSpace(req.SystemPrompt)

	messages := make([]*llmchatservice.ChatMsg, 0, 2)
	if systemPrompt != "" && convID == "" {
		messages = append(messages, &llmchatservice.ChatMsg{
			Role:    chatconsts.ChatMessageRoleSystem,
			Content: systemPrompt,
		})
	}
	messages = append(messages, &llmchatservice.ChatMsg{
		Role:    chatconsts.ChatMessageRoleUser,
		Content: trimmedMsg,
	})

	autoFillHistory := req.AutoFillHistory
	if convID != "" && !req.AutoFillHistory {
		// default to true when continuing an existing conversation
		autoFillHistory = true
	}

	chatReq := &llmchatservice.ChatReq{
		UserId:          req.UserId,
		ConversationId:  convID,
		LlmConfig:       llmCfg,
		Messages:        messages,
		AutoFillHistory: autoFillHistory,
	}

	chatResp, err := l.svcCtx.LlmChatRpc.Chat(l.ctx, chatReq)
	if err != nil {
		l.Logger.Errorf("LlmChatRpc.Chat error, configId=%d, userId=%d, err=%v", req.ConfigId, req.UserId, err)
		return nil, errors.Wrap(err, "chat service failed")
	}

	if chatResp == nil {
		return nil, errors.New("empty response from chat service")
	}

	respMessages := make([]types.TextChatMessage, 0, 1+len(chatResp.GetRespMsg()))

	for _, msg := range chatResp.GetRespMsg() {
		if msg == nil {
			continue
		}
		respMessages = append(respMessages, types.TextChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return &types.TextChatResp{
		ConversationId: chatResp.GetId(),
		Messages:       respMessages,
	}, nil
}

func buildChatLlmConfig(cfg *llmconfigservice.ChatConfig) *llmchatservice.LlmConfig {
	if cfg == nil {
		return nil
	}

	return &llmchatservice.LlmConfig{
		BaseUrl:           strings.TrimSpace(cfg.BaseUrl),
		ApiKey:            strings.TrimSpace(cfg.ApiKey),
		Model:             strings.TrimSpace(cfg.Model),
		Temperature:       cfg.Temperature,
		TopP:              cfg.TopP,
		TopK:              cfg.TopK,
		EnableThinking:    cfg.EnableThinking > 0,
		RepetitionPenalty: cfg.RepetitionPenalty,
		PresencePenalty:   cfg.PresencePenalty,
		MaxTokens:         cfg.MaxTokens,
		Seed:              cfg.Seed,
		EnableSearch:      cfg.EnableSearch > 0,
		ContentLength:     cfg.ContextLength,
	}
}
