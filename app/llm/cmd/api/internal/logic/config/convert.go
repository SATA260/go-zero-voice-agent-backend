package config

import (
	"go-zero-voice-agent/app/llm/cmd/api/internal/types"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/llmconfigservice"
)

func toRpcCreateConfigReq(req *types.CreateConfigReq) *llmconfigservice.CreateConfigReq {
	if req == nil {
		return nil
	}

	return &llmconfigservice.CreateConfigReq{
		Name:              req.Name,
		Description:       req.Description,
		UserId:            req.UserId,
		BaseUrl:           req.BaseUrl,
		ApiKey:            req.ApiKey,
		Model:             req.Model,
		Stream:            req.Stream,
		Temperature:       req.Temperature,
		TopP:              req.TopP,
		TopK:              req.TopK,
		EnableThinking:    req.EnableThinking,
		RepetitionPenalty: req.RepetitionPenalty,
		PresencePenalty:   req.PresencePenalty,
		MaxTokens:         req.MaxTokens,
		Seed:              req.Seed,
		EnableSearch:      req.EnableSearch,
		ContextLength:     req.ContextLength,
	}
}

func toRpcUpdateConfigReq(req *types.UpdateConfigReq) *llmconfigservice.UpdateConfigReq {
	if req == nil {
		return nil
	}

	return &llmconfigservice.UpdateConfigReq{
		Id:                req.Id,
		Name:              req.Name,
		Description:       req.Description,
		UserId:            req.UserId,
		BaseUrl:           req.BaseUrl,
		ApiKey:            req.ApiKey,
		Model:             req.Model,
		Stream:            req.Stream,
		Temperature:       req.Temperature,
		TopP:              req.TopP,
		TopK:              req.TopK,
		EnableThinking:    req.EnableThinking,
		RepetitionPenalty: req.RepetitionPenalty,
		PresencePenalty:   req.PresencePenalty,
		MaxTokens:         req.MaxTokens,
		Seed:              req.Seed,
		EnableSearch:      req.EnableSearch,
		ContextLength:     req.ContextLength,
	}
}

func toRpcDeleteConfigReq(id int64) *llmconfigservice.DeleteConfigReq {
	return &llmconfigservice.DeleteConfigReq{Id: id}
}

func toRpcGetConfigReq(id int64) *llmconfigservice.GetConfigReq {
	return &llmconfigservice.GetConfigReq{Id: id}
}

func toRpcListConfigReq(req *types.ListMyConfigReq) *llmconfigservice.ListConfigReq {
	if req == nil {
		return nil
	}

	return &llmconfigservice.ListConfigReq{
		PageQuery:    toRpcPageQuery(req.PageQuery),
		Filter: toRpcChatConfigFilter(req.Filter),
	}
}

func toRpcPageQuery(query types.PageQuery) *llmconfigservice.PageQuery {
	return &llmconfigservice.PageQuery{
		Page:     query.Page,
		PageSize: query.PageSize,
		OrderBy:  query.OrderBy,
	}
}

func toRpcChatConfigFilter(filter types.ChatConfigQueryFilter) *llmconfigservice.ListConfigFilter {
	return &llmconfigservice.ListConfigFilter{
		Id:                filter.Id,
		Name:              filter.Name,
		Description:       filter.Description,
		UserId:            filter.UserId,
	}
}

func toTypesChatConfig(cfg *llmconfigservice.ChatConfig) types.ChatConfig {
	if cfg == nil {
		return types.ChatConfig{}
	}

	return types.ChatConfig{
		Id:                cfg.Id,
		Name:              cfg.Name,
		Description:       cfg.Description,
		UserId:            cfg.UserId,
		BaseUrl:           cfg.BaseUrl,
		ApiKey:            cfg.ApiKey,
		Model:             cfg.Model,
		Stream:            cfg.Stream,
		Temperature:       cfg.Temperature,
		TopP:              cfg.TopP,
		TopK:              cfg.TopK,
		EnableThinking:    cfg.EnableThinking,
		RepetitionPenalty: cfg.RepetitionPenalty,
		PresencePenalty:   cfg.PresencePenalty,
		MaxTokens:         cfg.MaxTokens,
		Seed:              cfg.Seed,
		EnableSearch:      cfg.EnableSearch,
		ContextLength:     cfg.ContextLength,
	}
}
