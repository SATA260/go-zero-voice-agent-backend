package ttsconfigservicelogic

import (
	"context"

	"go-zero-voice-agent/app/voicechat/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/voicechatpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteTtsConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteTtsConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteTtsConfigLogic {
	return &DeleteTtsConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteTtsConfigLogic) DeleteTtsConfig(in *voicechatpb.DeleteTtsConfigRequest) (*voicechatpb.DeleteTtsConfigResponse, error) {
	data, err := l.svcCtx.TtsConfigModel.FindOne(l.ctx, in.Id)
	if err != nil {
		return nil, err
	}

	err = l.svcCtx.TtsConfigModel.DeleteSoft(l.ctx, nil, data)
	if err != nil {
		return nil, err
	}

	return &voicechatpb.DeleteTtsConfigResponse{Ok: true}, nil
}
