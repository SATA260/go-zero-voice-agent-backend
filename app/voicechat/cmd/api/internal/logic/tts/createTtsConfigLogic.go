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

type CreateTtsConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建TTS配置
func NewCreateTtsConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTtsConfigLogic {
	return &CreateTtsConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateTtsConfigLogic) CreateTtsConfig(req *types.CreateTtsConfigReq) (resp *types.CreateTtsConfigResp, err error) {
	if req.UserId <= 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	r, err := l.svcCtx.TtsConfigRpc.CreateTtsConfig(l.ctx, &ttsconfigservice.CreateTtsConfigRequest{
		UserId:    req.UserId,
		Provider:  req.Provider,
		AppId:     req.AppId,
		SecretId:  req.SecretId,
		SecretKey: req.SecretKey,
	})
	if err != nil {
		return nil, err
	}
	return &types.CreateTtsConfigResp{Id: r.Config.Id}, nil
}
