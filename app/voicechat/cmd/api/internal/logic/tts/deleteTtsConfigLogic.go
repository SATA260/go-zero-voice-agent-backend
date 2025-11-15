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

type DeleteTtsConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除TTS配置
func NewDeleteTtsConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteTtsConfigLogic {
	return &DeleteTtsConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteTtsConfigLogic) DeleteTtsConfig(req *types.DeleteTtsConfigReq) (resp *types.DeleteTtsConfigResp, err error) {
	if req.UserId <= 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	// ownership check
	gr, err := l.svcCtx.TtsConfigRpc.GetTtsConfig(l.ctx, &ttsconfigservice.GetTtsConfigRequest{Id: req.Id})
	if err != nil {
		return nil, err
	}
	if gr.Config.UserId != req.UserId {
		return nil, xerr.NewErrCode(xerr.USER_PERMISSION_DENIED_ERROR)
	}
	_, err = l.svcCtx.TtsConfigRpc.DeleteTtsConfig(l.ctx, &ttsconfigservice.DeleteTtsConfigRequest{Id: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.DeleteTtsConfigResp{}, nil
}
