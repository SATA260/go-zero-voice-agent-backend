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

type ListTtsConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 分页获取TTS配置列表
func NewListTtsConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTtsConfigLogic {
	return &ListTtsConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListTtsConfigLogic) ListTtsConfig(req *types.ListTtsConfigReq) (resp *types.ListTtsConfigResp, err error) {
	if req.UserId <= 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	r, err := l.svcCtx.TtsConfigRpc.ListTtsConfig(l.ctx, &ttsconfigservice.ListTtsConfigRequest{
		Page:   &ttsconfigservice.PageQuery{Page: req.Page, PageSize: req.PageSize},
		UserId: req.UserId,
	})
	if err != nil {
		return nil, err
	}

	list := make([]types.GetTtsConfigResp, 0, len(r.Configs))
	for _, cfg := range r.Configs {
		list = append(list, types.GetTtsConfigResp{
			Id:        cfg.Id,
			UserId:    cfg.UserId,
			Provider:  cfg.Provider,
			AppId:     cfg.AppId,
			SecretId:  cfg.SecretId,
			SecretKey: cfg.SecretKey,
		})
	}
	return &types.ListTtsConfigResp{ConfigList: list, Total: r.Total}, nil
}
