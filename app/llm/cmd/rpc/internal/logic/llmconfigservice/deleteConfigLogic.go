package llmconfigservicelogic

import (
	"context"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteConfigLogic {
	return &DeleteConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteConfigLogic) DeleteConfig(in *pb.DeleteConfigReq) (*pb.DeleteConfigResp, error) {
	_, err := l.svcCtx.ChatConfigModel.FindOne(l.ctx, in.Id)
    if err != nil {
        if err == model.ErrNotFound {
            return nil, errors.Wrapf(model.ErrNotFound, "config not found, id: %d", in.Id)
        }
        return nil, errors.Wrapf(err, "FindOne config failed, id: %d", in.Id)
    }

    err = l.svcCtx.ChatConfigModel.Delete(l.ctx, nil, in.Id)
    if err != nil {
        return nil, errors.Wrapf(err, "Delete config failed, id: %d", in.Id)
    }

    return &pb.DeleteConfigResp{}, nil
}
