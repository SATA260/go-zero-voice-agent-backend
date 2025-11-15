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

type CreateAsrConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建ASR配置
func NewCreateAsrConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAsrConfigLogic {
	return &CreateAsrConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateAsrConfigLogic) CreateAsrConfig(req *types.CreateAsrConfigReq) (resp *types.CreateAsrConfigResp, err error) {
	if req.UserId <= 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	r, err := l.svcCtx.AsrConfigRpc.CreateAsrConfig(l.ctx, &asrconfigservice.CreateAsrConfigRequest{
		UserId:    req.UserId,
		Provider:  req.Provider,
		AppId:     req.AppId,
		SecretId:  req.SecretId,
		SecretKey: req.SecretKey,
		Language:  req.Language,
	})
	if err != nil {
		return nil, err
	}

	return &types.CreateAsrConfigResp{Id: r.Config.Id}, nil
}
