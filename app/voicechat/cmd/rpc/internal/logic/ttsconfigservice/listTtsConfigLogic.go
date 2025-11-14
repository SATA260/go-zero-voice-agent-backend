package ttsconfigservicelogic

import (
	"context"

	"go-zero-voice-agent/app/voicechat/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/pb"
	"go-zero-voice-agent/app/voicechat/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListTtsConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListTtsConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTtsConfigLogic {
	return &ListTtsConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListTtsConfigLogic) ListTtsConfig(in *pb.ListTtsConfigRequest) (*pb.ListTtsConfigResponse, error) {
	builder := l.svcCtx.TtsConfigModel.SelectBuilder()
	if in.UserId != 0 {
		builder = builder.Where("user_id = ?", in.UserId)
	}

	var configs []*model.TtsConfig
	var total int64
	var err error

	if in.Page != nil && in.Page.PageSize > 0 {
		configs, total, err = l.svcCtx.TtsConfigModel.FindPageListByPageWithTotal(l.ctx, builder, in.Page.Page, in.Page.PageSize, in.Page.OrderBy)
	} else {
		configs, err = l.svcCtx.TtsConfigModel.FindAll(l.ctx, builder, "")
		if err == nil {
			total = int64(len(configs))
		}
	}
	if err != nil {
		return nil, err
	}

	respList := make([]*pb.TtsConfig, 0, len(configs))
	for _, c := range configs {
		respList = append(respList, &pb.TtsConfig{
			Id:        c.Id,
			UserId:    c.UserId.Int64,
			Provider:  c.Provider.String,
			AppId:     c.AppId.String,
			SecretId:  c.SecretId.String,
			SecretKey: c.SecretKey.String,
		})
	}

	return &pb.ListTtsConfigResponse{
		Configs: respList,
		Total:   total,
	}, nil
}
