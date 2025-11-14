package asrconfigservicelogic

import (
	"context"
	"database/sql"

	"go-zero-voice-agent/app/voicechat/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/pb"
	"go-zero-voice-agent/app/voicechat/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateAsrConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateAsrConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAsrConfigLogic {
	return &CreateAsrConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateAsrConfigLogic) CreateAsrConfig(in *pb.CreateAsrConfigRequest) (*pb.CreateAsrConfigResponse, error) {
	data := &model.AsrConfig{
		UserId:    sql.NullInt64{Int64: in.Config.UserId, Valid: in.Config.UserId != 0},
		Provider:  sql.NullString{String: in.Config.Provider, Valid: in.Config.Provider != ""},
		AppId:     sql.NullString{String: in.Config.AppId, Valid: in.Config.AppId != ""},
		SecretId:  sql.NullString{String: in.Config.SecretId, Valid: in.Config.SecretId != ""},
		SecretKey: sql.NullString{String: in.Config.SecretKey, Valid: in.Config.SecretKey != ""},
		Language:  sql.NullString{String: in.Config.Language, Valid: in.Config.Language != ""},
	}

	res, err := l.svcCtx.AsrConfigModel.Insert(l.ctx, nil, data)
	if err != nil {
		return nil, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	data.Id = lastId

	return &pb.CreateAsrConfigResponse{
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
