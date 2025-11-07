// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"context"

	"go-zero-voice-agent/app/llm/cmd/api/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMyConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 分页查询我的配置
func NewListMyConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMyConfigLogic {
	return &ListMyConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListMyConfigLogic) ListMyConfig(req *types.ListMyConfigReq) (resp *types.ListMyConfigResp, err error) {
	listResp, err := l.svcCtx.LlmConfigRpc.ListConfig(l.ctx, toRpcListConfigReq(req))
	if err != nil {
		return nil, err
	}

	configs := make([]types.ChatConfig, 0, len(listResp.Configs))
	for _, cfg := range listResp.Configs {
		configs = append(configs, toTypesChatConfig(cfg))
	}

	return &types.ListMyConfigResp{
		Total:   listResp.Total,
		Configs: configs,
	}, nil
}
