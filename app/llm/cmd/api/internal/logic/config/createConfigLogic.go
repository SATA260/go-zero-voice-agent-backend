// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"context"

	"go-zero-voice-agent/app/llm/cmd/api/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建配置
func NewCreateConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateConfigLogic {
	return &CreateConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateConfigLogic) CreateConfig(req *types.CreateConfigReq) (resp *types.CreateConfigResp, err error) {
	createResp, err := l.svcCtx.LlmConfigRpc.CreateConfig(l.ctx, toRpcCreateConfigReq(req))
	if err != nil {
		return nil, err
	}

	return &types.CreateConfigResp{Id: createResp.Id}, nil
}
