package llmconfigservicelogic

import (
	"context"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"
	"go-zero-voice-agent/pkg/tool"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConfigLogic {
	return &GetConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetConfigLogic) GetConfig(in *pb.GetConfigReq) (*pb.GetConfigResp, error) {
	res, err := l.svcCtx.ChatConfigModel.FindOne(l.ctx, in.Id)
    if err != nil {
        if err == model.ErrNotFound {
            return nil, errors.Wrapf(model.ErrNotFound, "config not found, id: %d", in.Id)
        }
        return nil, errors.Wrapf(err, "FindOne config failed, id: %d", in.Id)
    }

    return &pb.GetConfigResp{
        Config: chatConfigToPb(res),
    }, nil
}

func chatConfigToPb(cfg *model.ChatConfig) *pb.ChatConfig {
    if cfg == nil {
        return nil
    }
    return &pb.ChatConfig{
		Id: 			   cfg.Id,
        Name:              cfg.Name,
        Description:       tool.NullStringToString(cfg.Description),
        UserId:            tool.NullInt64ToInt64(cfg.UserId),
        BaseUrl:           tool.NullStringToString(cfg.BaseUrl),
        ApiKey:            tool.NullStringToString(cfg.ApiKey),
        Model:             tool.NullStringToString(cfg.Model),
        Stream:            tool.NullInt64ToInt64(cfg.Stream),
        Temperature:       tool.NullFloat64ToFloat64(cfg.Temperature),
        TopP:              tool.NullFloat64ToFloat64(cfg.TopP),
        TopK:              tool.NullInt64ToInt64(cfg.TopK),
        EnableThinking:    tool.NullInt64ToInt64(cfg.EnableThinking),
        RepetitionPenalty: tool.NullFloat64ToFloat64(cfg.RepetitionPenalty),
        PresencePenalty:   tool.NullFloat64ToFloat64(cfg.PresencePenalty),
        MaxTokens:         tool.NullInt64ToInt64(cfg.MaxTokens),
        Seed:              tool.NullInt64ToInt64(cfg.Seed),
        EnableSearch:      tool.NullInt64ToInt64(cfg.EnableSearch),
        ContextLength:     tool.NullInt64ToInt64(cfg.ContextLength),
    }
}