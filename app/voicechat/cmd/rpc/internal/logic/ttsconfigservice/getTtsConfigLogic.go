package ttsconfigservicelogic

import (
	"context"

	"go-zero-voice-agent/app/voicechat/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/voicechatpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTtsConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTtsConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTtsConfigLogic {
	return &GetTtsConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetTtsConfigLogic) GetTtsConfig(in *voicechatpb.GetTtsConfigRequest) (*voicechatpb.GetTtsConfigResponse, error) {
	data, err := l.svcCtx.TtsConfigModel.FindOne(l.ctx, in.Id)
	if err != nil {
		return nil, err
	}

	return &voicechatpb.GetTtsConfigResponse{
		Config: &voicechatpb.TtsConfig{
			Id:        data.Id,
			UserId:    data.UserId.Int64,
			Provider:  data.Provider.String,
			AppId:     data.AppId.String,
			SecretId:  data.SecretId.String,
			SecretKey: data.SecretKey.String,
		},
	}, nil
}
