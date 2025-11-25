package chatsessionservicelogic

import (
	"context"
	"strings"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetChatSessionByConvIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetChatSessionByConvIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetChatSessionByConvIdLogic {
	return &GetChatSessionByConvIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetChatSessionByConvIdLogic) GetChatSessionByConvId(in *pb.GetChatSessionByConvIdReq) (*pb.GetChatSessionResp, error) {
	if in == nil {
		return nil, errors.New("invalid request")
	}

	convID := strings.TrimSpace(in.GetConvId())
	if convID == "" {
		return nil, errors.New("conv_id is required")
	}

	session, err := l.svcCtx.ChatSessionModel.FindOneByConvId(l.ctx, convID)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, errors.Wrapf(model.ErrNotFound, "chat session not found, conv_id: %s", convID)
		}
		return nil, errors.Wrapf(err, "find chat session failed, conv_id: %s", convID)
	}

	return &pb.GetChatSessionResp{Session: chatSessionToPb(session)}, nil
}
