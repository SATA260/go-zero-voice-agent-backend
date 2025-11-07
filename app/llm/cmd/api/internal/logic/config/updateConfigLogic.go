// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"context"

	"go-zero-voice-agent/app/llm/cmd/api/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/api/internal/types"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/llmchatservice"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新配置
func NewUpdateConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateConfigLogic {
	return &UpdateConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateConfigLogic) UpdateConfig(req *types.UpdateConfigReq) (resp *types.UpdateConfigResp, err error) {
	if rpcResp, err := l.svcCtx.LlmConfigRpc.GetConfig(l.ctx, &llmchatservice.GetConfigReq{Id: req.Id}); err != nil {
		return nil, err
	} else if rpcResp.Config.UserId != req.UserId {
		return nil, errors.New("Not authorized to update this config")
	}

	if _, err = l.svcCtx.LlmConfigRpc.UpdateConfig(l.ctx, toRpcUpdateConfigReq(req)); err != nil {
		return nil, err
	}

	return &types.UpdateConfigResp{}, nil
}
