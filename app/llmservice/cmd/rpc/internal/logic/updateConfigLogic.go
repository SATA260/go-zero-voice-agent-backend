package logic

import (
	"context"

	"go-zero-voice-agent/app/llmservice/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llmservice/cmd/rpc/pb"
	"go-zero-voice-agent/app/llmservice/model"
	"go-zero-voice-agent/pkg/tool"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateConfigLogic {
	return &UpdateConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateConfigLogic) UpdateConfig(in *pb.UpdateConfigReq) (*pb.UpdateConfigResp, error) {
	_, err := l.svcCtx.ChatConfigModel.FindOne(l.ctx, in.Id)
    if err != nil {
        if err == model.ErrNotFound {
            return nil, errors.Wrapf(model.ErrNotFound, "config not found, id: %d", in.Id)
        }
        return nil, errors.Wrapf(err, "FindOne config failed, id: %d", in.Id)
    }

    chatConfig := updateConfigReqToModel(in)
    // err = l.svcCtx.ChatConfigModel.Update(l.ctx, chatConfig)
	_, err = l.svcCtx.ChatConfigModel.Update(l.ctx, nil, chatConfig)
    if err != nil {
        return nil, errors.Wrapf(err, "Update chat config failed. req: %+v", in)
    }
	

    return &pb.UpdateConfigResp{}, nil
}

func updateConfigReqToModel(cfg *pb.UpdateConfigReq) *model.ChatConfig {
    return &model.ChatConfig{
		Id: 			   cfg.Id,	
        Name:              cfg.Name,
        Description:       tool.StringToNullString(cfg.Description),
        UserId:            tool.Int64ToNullInt64(cfg.UserId),
        BaseUrl:           tool.StringToNullString(cfg.BaseUrl),
        ApiKey:            tool.StringToNullString(cfg.ApiKey),
        Model:             tool.StringToNullString(cfg.Model),
        Stream:            tool.Int64ToNullInt64(cfg.Stream),
        Temperature:       tool.Float64ToNullFloat64(cfg.Temperature),
        TopP:              tool.Float64ToNullFloat64(cfg.TopP),
        TopK:              tool.Int64ToNullInt64(cfg.TopK),
        EnableThinking:    tool.Int64ToNullInt64(cfg.EnableThinking),
        RepetitionPenalty: tool.Float64ToNullFloat64(cfg.RepetitionPenalty),
        PresencePenalty:   tool.Float64ToNullFloat64(cfg.PresencePenalty),
        MaxTokens:         tool.Int64ToNullInt64(cfg.MaxTokens),
        Seed:              tool.Int64ToNullInt64(cfg.Seed),
        EnableSearch:      tool.Int64ToNullInt64(cfg.EnableSearch),
        ContextLength:     tool.Int64ToNullInt64(cfg.ContextLength),
    }
}