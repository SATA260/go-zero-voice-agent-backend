package ttsconfigservicelogic

import (
	"context"

	"go-zero-voice-agent/app/voicechat/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/pb"

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

func (l *GetTtsConfigLogic) GetTtsConfig(in *pb.GetTtsConfigRequest) (*pb.GetTtsConfigResponse, error) {
	data, err := l.svcCtx.TtsConfigModel.FindOne(l.ctx, in.Id)
	if err != nil {
		return nil, err
	}

	return &pb.GetTtsConfigResponse{
		Config: &pb.TtsConfig{
			Id:        data.Id,
			UserId:    data.UserId.Int64,
			Provider:  data.Provider.String,
			AppId:     data.AppId.String,
			SecretId:  data.SecretId.String,
			SecretKey: data.SecretKey.String,
		},
	}, nil
}
