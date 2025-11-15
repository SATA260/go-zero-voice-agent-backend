// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package asr

import (
	"context"

	"go-zero-voice-agent/app/voicechat/cmd/api/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/types"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/client/asrconfigservice"
	"go-zero-voice-agent/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAsrConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取ASR配置详情
func NewGetAsrConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAsrConfigLogic {
	return &GetAsrConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAsrConfigLogic) GetAsrConfig(req *types.GetAsrConfigReq) (resp *types.GetAsrConfigResp, err error) {
	r, err := l.svcCtx.AsrConfigRpc.GetAsrConfig(l.ctx, &asrconfigservice.GetAsrConfigRequest{Id: req.Id})
	if err != nil {
		return nil, err
	}
	cfg := r.Config
	if req.UserId <= 0 || cfg.UserId != req.UserId {
		return nil, xerr.NewErrCode(xerr.USER_PERMISSION_DENIED_ERROR)
	}
	return &types.GetAsrConfigResp{
		Id:        cfg.Id,
		UserId:    cfg.UserId,
		Provider:  cfg.Provider,
		AppId:     cfg.AppId,
		SecretId:  cfg.SecretId,
		SecretKey: cfg.SecretKey,
		Language:  cfg.Language,
	}, nil
}
