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
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
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
	if err := l.validReq(req); err != nil {
		return nil, err
	}

	fetchLlmConfig, err := l.fetchAndValidLlmConfig(req.ConfigId, req.UserId)
	if err != nil {
		return nil, err
	}

	llmCfg := l.buildChatLlmConfig(fetchLlmConfig)
	convID := strings.TrimSpace(req.ConversationId)
	systemPrompt := strings.TrimSpace(req.SystemPrompt)
	trimmedMsg := strings.TrimSpace(req.Message)
	var messages []*llmchatservice.ChatMsg
	if systemPrompt != "" && convID == "" {
		messages = l.createChatMsgs(systemPrompt, trimmedMsg)
	} else {
		messages = l.createChatMsgs("", trimmedMsg)
	}

	// 发起对话请求
	chatReq := &llmchatservice.ChatReq{
		UserId:          req.UserId,
		ConversationId:  convID,
		LlmConfig:       llmCfg,
		Messages:        messages,
		AutoFillHistory: req.AutoFillHistory,
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

		var toolCalls []types.ToolCall
		if len(msg.ToolCalls) > 0 {
			toolCalls = make([]types.ToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				toolCalls[i] = types.ToolCall{
					Info: types.ToolCallInfo{
						Id:                   tc.Info.Id,
						Name:                 tc.Info.Name,
						ArgumentsJson:        tc.Info.ArgumentsJson,
						Scope:                tc.Info.Scope,
						RequiresConfirmation: tc.Info.RequiresConfirmation,
					},
					Status: tc.Status,
					Result: tc.Result,
					Error:  tc.Error,
				}
			}
		}

		respMessages = append(respMessages, types.TextChatMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCalls:  toolCalls,
			ToolCallId: msg.ToolCallId,
		})
	}

	// 直接使用http response返回响应消息
	return &types.TextChatResp{
		ConversationId: chatResp.GetConversationId(),
		Messages:       respMessages,
	}, nil
}

func (l *TextChatLogic) TextChatStream(req *types.TextChatReq) (pb.LlmChatService_ChatStreamClient, error) {
	if err := l.validReq(req); err != nil {
		return nil, err
	}

	fetchLlmConfig, err := l.fetchAndValidLlmConfig(req.ConfigId, req.UserId)
	if err != nil {
		return nil, err
	}

	llmCfg := l.buildChatLlmConfig(fetchLlmConfig)
	convID := strings.TrimSpace(req.ConversationId)
	systemPrompt := strings.TrimSpace(req.SystemPrompt)
	trimmedMsg := strings.TrimSpace(req.Message)
	var messages []*llmchatservice.ChatMsg
	if systemPrompt != "" && convID == "" {
		messages = l.createChatMsgs(systemPrompt, trimmedMsg)
	} else {
		messages = l.createChatMsgs("", trimmedMsg)
	}

	// 发起流式对话请求，并返回流式响应客户端
	chatStreamClient, err := l.svcCtx.LlmChatRpc.ChatStream(l.ctx, &llmchatservice.ChatStreamReq{
		UserId:          req.UserId,
		ConversationId:  convID,
		LlmConfig:       llmCfg,
		Messages:        messages,
		AutoFillHistory: req.AutoFillHistory,
	})
	return chatStreamClient, err
}

// 校验请求参数
func (l *TextChatLogic) validReq(req *types.TextChatReq) error {
	if req == nil {
		return errors.New("invalid request")
	}

	trimmedMsg := strings.TrimSpace(req.Message)
	if trimmedMsg == "" {
		return errors.New("message cannot be empty")
	}

	if req.ConfigId <= 0 {
		return errors.New("configId must be greater than 0")
	}

	return nil
}

// 获取并校验LLM配置
func (l *TextChatLogic) fetchAndValidLlmConfig(configId, userId int64) (*llmconfigservice.ChatConfig, error) {
	cfgResp, err := l.svcCtx.LlmConfigRpc.GetConfig(l.ctx, &llmconfigservice.GetConfigReq{Id: configId})
	if err != nil {
		l.Logger.Errorf("LlmConfigRpc.GetConfig error, configId=%d, err=%v", configId, err)
		return nil, errors.Wrap(err, "failed to fetch config")
	}

	cfg := cfgResp.GetConfig()
	if cfg == nil {
		return nil, errors.New("config not found")
	}

	if cfg.UserId != userId {
		return nil, errors.New("not authorized to use this config")
	}

	llmCfg := l.buildChatLlmConfig(cfg)
	if llmCfg == nil {
		return nil, errors.New("invalid llm config")
	}

	if llmCfg.ApiKey == "" || llmCfg.Model == "" {
		return nil, errors.New("llm config missing api key or model")
	}

	return cfg, nil
}

// 构建LLM配置
func (l *TextChatLogic) buildChatLlmConfig(cfg *llmconfigservice.ChatConfig) *llmchatservice.LlmConfig {
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

// 创建聊天消息列表
func (l *TextChatLogic) createChatMsgs(systemPrompt string, message string) []*llmchatservice.ChatMsg {
	messages := make([]*llmchatservice.ChatMsg, 0, 2)
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, &llmchatservice.ChatMsg{
			Role:    chatconsts.ChatMessageRoleSystem,
			Content: systemPrompt,
		})
	}
	messages = append(messages, &llmchatservice.ChatMsg{
		Role:    chatconsts.ChatMessageRoleUser,
		Content: message,
	})

	return messages
}
