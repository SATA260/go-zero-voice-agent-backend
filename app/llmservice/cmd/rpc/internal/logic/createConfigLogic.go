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

type CreateConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateConfigLogic {
	return &CreateConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateConfigLogic) CreateConfig(in *pb.CreateConfigReq) (*pb.CreateConfigResp, error) {
	chatConfig := createConfigReqToModel(in)
	result, err := l.svcCtx.ChatConfigModel.Insert(l.ctx, nil, chatConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "Insert chat config failed. req: %+v", in)
	}

	id, _ := result.LastInsertId()
	return &pb.CreateConfigResp{Id: id}, nil
}


func createConfigReqToModel(cfg *pb.CreateConfigReq) *model.ChatConfig {
    return &model.ChatConfig{
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