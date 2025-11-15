package asrconfigservicelogic

import (
	"context"
	"database/sql"

	"go-zero-voice-agent/app/voicechat/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/voicechatpb"
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

func (l *CreateAsrConfigLogic) CreateAsrConfig(in *voicechatpb.CreateAsrConfigRequest) (*voicechatpb.CreateAsrConfigResponse, error) {
	data := &model.AsrConfig{
		UserId:    sql.NullInt64{Int64: in.UserId, Valid: in.UserId != 0},
		Provider:  sql.NullString{String: in.Provider, Valid: in.Provider != ""},
		AppId:     sql.NullString{String: in.AppId, Valid: in.AppId != ""},
		SecretId:  sql.NullString{String: in.SecretId, Valid: in.SecretId != ""},
		SecretKey: sql.NullString{String: in.SecretKey, Valid: in.SecretKey != ""},
		Language:  sql.NullString{String: in.Language, Valid: in.Language != ""},
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

	return &voicechatpb.CreateAsrConfigResponse{
		Config: &voicechatpb.AsrConfig{
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
