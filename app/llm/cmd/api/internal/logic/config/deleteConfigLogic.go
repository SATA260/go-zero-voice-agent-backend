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

type DeleteConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除配置
func NewDeleteConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteConfigLogic {
	return &DeleteConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteConfigLogic) DeleteConfig(req *types.DeleteConfigReq) (resp *types.DeleteConfigResp, err error) {
	if rpcResp, err := l.svcCtx.LlmConfigRpc.GetConfig(l.ctx, &llmchatservice.GetConfigReq{Id: req.Id}); err != nil {
		return nil, err
	} else if rpcResp.Config.UserId != req.UserId {
		return nil, errors.New("Not authorized to delete this config")
	}
	

	if _, err = l.svcCtx.LlmConfigRpc.DeleteConfig(l.ctx, toRpcDeleteConfigReq(req.Id)); err != nil {
		return nil, err
	}

	return &types.DeleteConfigResp{}, nil
}
