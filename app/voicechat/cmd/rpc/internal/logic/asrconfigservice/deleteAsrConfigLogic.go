package asrconfigservicelogic

import (
	"context"

	"go-zero-voice-agent/app/voicechat/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/voicechatpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteAsrConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteAsrConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteAsrConfigLogic {
	return &DeleteAsrConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteAsrConfigLogic) DeleteAsrConfig(in *voicechatpb.DeleteAsrConfigRequest) (*voicechatpb.DeleteAsrConfigResponse, error) {
	data, err := l.svcCtx.AsrConfigModel.FindOne(l.ctx, in.Id)
	if err != nil {
		return nil, err
	}

	err = l.svcCtx.AsrConfigModel.DeleteSoft(l.ctx, nil, data)
	if err != nil {
		return nil, err
	}

	return &voicechatpb.DeleteAsrConfigResponse{Ok: true}, nil
}
