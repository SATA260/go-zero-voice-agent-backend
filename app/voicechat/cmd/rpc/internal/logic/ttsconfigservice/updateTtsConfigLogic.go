package ttsconfigservicelogic

import (
	"context"
	"database/sql"

	"go-zero-voice-agent/app/voicechat/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/pb"
	"go-zero-voice-agent/app/voicechat/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateTtsConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateTtsConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateTtsConfigLogic {
	return &UpdateTtsConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateTtsConfigLogic) UpdateTtsConfig(in *pb.UpdateTtsConfigRequest) (*pb.UpdateTtsConfigResponse, error) {
	data := &model.TtsConfig{
		Id:        in.Config.Id,
		UserId:    sql.NullInt64{Int64: in.Config.UserId, Valid: in.Config.UserId != 0},
		Provider:  sql.NullString{String: in.Config.Provider, Valid: in.Config.Provider != ""},
		AppId:     sql.NullString{String: in.Config.AppId, Valid: in.Config.AppId != ""},
		SecretId:  sql.NullString{String: in.Config.SecretId, Valid: in.Config.SecretId != ""},
		SecretKey: sql.NullString{String: in.Config.SecretKey, Valid: in.Config.SecretKey != ""},
	}

	_, err := l.svcCtx.TtsConfigModel.Update(l.ctx, nil, data)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateTtsConfigResponse{
		Config: in.Config,
	}, nil
}
