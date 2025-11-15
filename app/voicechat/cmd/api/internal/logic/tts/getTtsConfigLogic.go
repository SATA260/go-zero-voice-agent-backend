// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package tts

import (
	"context"

	"go-zero-voice-agent/app/voicechat/cmd/api/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/types"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/client/ttsconfigservice"
	"go-zero-voice-agent/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTtsConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取TTS配置详情
func NewGetTtsConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTtsConfigLogic {
	return &GetTtsConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTtsConfigLogic) GetTtsConfig(req *types.GetTtsConfigReq) (resp *types.GetTtsConfigResp, err error) {
	r, err := l.svcCtx.TtsConfigRpc.GetTtsConfig(l.ctx, &ttsconfigservice.GetTtsConfigRequest{Id: req.Id})
	if err != nil {
		return nil, err
	}
	cfg := r.Config
	if req.UserId <= 0 || cfg.UserId != req.UserId {
		return nil, xerr.NewErrCode(xerr.USER_PERMISSION_DENIED_ERROR)
	}
	return &types.GetTtsConfigResp{
		Id:        cfg.Id,
		UserId:    cfg.UserId,
		Provider:  cfg.Provider,
		AppId:     cfg.AppId,
		SecretId:  cfg.SecretId,
		SecretKey: cfg.SecretKey,
	}, nil
}
