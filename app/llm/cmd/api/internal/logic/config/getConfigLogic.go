// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"context"

	"go-zero-voice-agent/app/llm/cmd/api/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取配置详情
func NewGetConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConfigLogic {
	return &GetConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetConfigLogic) GetConfig(req *types.GetConfigReq) (resp *types.GetConfigResp, err error) {
	configResp, err := l.svcCtx.LlmConfigRpc.GetConfig(l.ctx, toRpcGetConfigReq(req.Id))
	if err != nil {
		return nil, err
	}

	return &types.GetConfigResp{Config: toTypesChatConfig(configResp.Config)}, nil
}
