package asrconfigservicelogic

import (
	"context"

	"go-zero-voice-agent/app/voicechat/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAsrConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAsrConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAsrConfigLogic {
	return &GetAsrConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetAsrConfigLogic) GetAsrConfig(in *pb.GetAsrConfigRequest) (*pb.GetAsrConfigResponse, error) {
	data, err := l.svcCtx.AsrConfigModel.FindOne(l.ctx, in.Id)
	if err != nil {
		return nil, err
	}

	return &pb.GetAsrConfigResponse{
		Config: &pb.AsrConfig{
			Id:        data.Id,
			UserId:    data.UserId.Int64,
			Provider:  data.Provider.String,
			AppId:     data.AppId.String,
			SecretId:  data.SecretId.String,
			SecretKey: data.SecretKey.String,
			Language:  data.Language.String,
		},
	}, nil
}
