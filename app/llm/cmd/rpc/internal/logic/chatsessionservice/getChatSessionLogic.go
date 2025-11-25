package chatsessionservicelogic

import (
	"context"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetChatSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetChatSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetChatSessionLogic {
	return &GetChatSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetChatSessionLogic) GetChatSession(in *pb.GetChatSessionReq) (*pb.GetChatSessionResp, error) {
	if in == nil {
		return nil, errors.New("invalid request")
	}

	session, err := l.svcCtx.ChatSessionModel.FindOne(l.ctx, in.GetId())
	if err != nil {
		if err == model.ErrNotFound {
			return nil, errors.Wrapf(model.ErrNotFound, "chat session not found, id: %d", in.GetId())
		}
		return nil, errors.Wrapf(err, "find chat session failed, id: %d", in.GetId())
	}

	return &pb.GetChatSessionResp{Session: chatSessionToPb(session)}, nil
}
