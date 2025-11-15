package ttsconfigservicelogic

import (
	"context"
	"database/sql"

	"go-zero-voice-agent/app/voicechat/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/voicechatpb"
	"go-zero-voice-agent/app/voicechat/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateTtsConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateTtsConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTtsConfigLogic {
	return &CreateTtsConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateTtsConfigLogic) CreateTtsConfig(in *voicechatpb.CreateTtsConfigRequest) (*voicechatpb.CreateTtsConfigResponse, error) {
	data := &model.TtsConfig{
		UserId:    sql.NullInt64{Int64: in.UserId, Valid: in.UserId != 0},
		Provider:  sql.NullString{String: in.Provider, Valid: in.Provider != ""},
		AppId:     sql.NullString{String: in.AppId, Valid: in.AppId != ""},
		SecretId:  sql.NullString{String: in.SecretId, Valid: in.SecretId != ""},
		SecretKey: sql.NullString{String: in.SecretKey, Valid: in.SecretKey != ""},
	}

	res, err := l.svcCtx.TtsConfigModel.Insert(l.ctx, nil, data)
	if err != nil {
		return nil, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	data.Id = lastId

	return &voicechatpb.CreateTtsConfigResponse{
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
