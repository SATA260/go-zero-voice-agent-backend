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

type ListAsrConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 分页获取ASR配置列表
func NewListAsrConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAsrConfigLogic {
	return &ListAsrConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListAsrConfigLogic) ListAsrConfig(req *types.ListAsrConfigReq) (resp *types.ListAsrConfigResp, err error) {
	if req.UserId <= 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	r, err := l.svcCtx.AsrConfigRpc.ListAsrConfig(l.ctx, &asrconfigservice.ListAsrConfigRequest{
		Page:   &asrconfigservice.PageQuery{Page: req.Page, PageSize: req.PageSize},
		UserId: req.UserId,
	})
	if err != nil {
		return nil, err
	}

	list := make([]types.GetAsrConfigResp, 0, len(r.Configs))
	for _, cfg := range r.Configs {
		list = append(list, types.GetAsrConfigResp{
			Id:        cfg.Id,
			UserId:    cfg.UserId,
			Provider:  cfg.Provider,
			AppId:     cfg.AppId,
			SecretId:  cfg.SecretId,
			SecretKey: cfg.SecretKey,
			Language:  cfg.Language,
		})
	}
	return &types.ListAsrConfigResp{ConfigList: list, Total: r.Total}, nil
}
