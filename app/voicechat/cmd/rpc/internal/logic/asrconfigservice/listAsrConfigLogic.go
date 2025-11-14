package asrconfigservicelogic

import (
	"context"

	"go-zero-voice-agent/app/voicechat/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/pb"
	"go-zero-voice-agent/app/voicechat/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAsrConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAsrConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAsrConfigLogic {
	return &ListAsrConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListAsrConfigLogic) ListAsrConfig(in *pb.ListAsrConfigRequest) (*pb.ListAsrConfigResponse, error) {
	builder := l.svcCtx.AsrConfigModel.SelectBuilder()
	if in.UserId != 0 {
		builder = builder.Where("user_id = ?", in.UserId)
	}

	var configs []*model.AsrConfig
	var total int64
	var err error

	if in.Page != nil && in.Page.PageSize > 0 {
		configs, total, err = l.svcCtx.AsrConfigModel.FindPageListByPageWithTotal(l.ctx, builder, in.Page.Page, in.Page.PageSize, in.Page.OrderBy)
	} else {
		configs, err = l.svcCtx.AsrConfigModel.FindAll(l.ctx, builder, "")
		if err == nil {
			total = int64(len(configs))
		}
	}
	if err != nil {
		return nil, err
	}

	respList := make([]*pb.AsrConfig, 0, len(configs))
	for _, c := range configs {
		respList = append(respList, &pb.AsrConfig{
			Id:        c.Id,
			UserId:    c.UserId.Int64,
			Provider:  c.Provider.String,
			AppId:     c.AppId.String,
			SecretId:  c.SecretId.String,
			SecretKey: c.SecretKey.String,
			Language:  c.Language.String,
		})
	}

	return &pb.ListAsrConfigResponse{
		Configs: respList,
		Total:   total,
	}, nil
}
