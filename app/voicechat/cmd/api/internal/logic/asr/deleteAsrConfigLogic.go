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

type DeleteAsrConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除ASR配置
func NewDeleteAsrConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteAsrConfigLogic {
	return &DeleteAsrConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteAsrConfigLogic) DeleteAsrConfig(req *types.DeleteAsrConfigReq) (resp *types.DeleteAsrConfigResp, err error) {
	if req.UserId <= 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	// ownership check
	gr, err := l.svcCtx.AsrConfigRpc.GetAsrConfig(l.ctx, &asrconfigservice.GetAsrConfigRequest{Id: req.Id})
	if err != nil {
		return nil, err
	}
	if gr.Config.UserId != req.UserId {
		return nil, xerr.NewErrCode(xerr.USER_PERMISSION_DENIED_ERROR)
	}
	_, err = l.svcCtx.AsrConfigRpc.DeleteAsrConfig(l.ctx, &asrconfigservice.DeleteAsrConfigRequest{Id: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.DeleteAsrConfigResp{}, nil
}
