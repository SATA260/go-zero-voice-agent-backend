package chatsessionservicelogic

import (
	"context"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteChatSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteChatSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteChatSessionLogic {
	return &DeleteChatSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteChatSessionLogic) DeleteChatSession(in *pb.DeleteChatSessionReq) (*pb.DeleteChatSessionResp, error) {
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

	if err := l.svcCtx.ChatSessionModel.DeleteSoft(l.ctx, nil, session); err != nil {
		return nil, errors.Wrapf(err, "delete chat session failed, id: %d", in.GetId())
	}

	return &pb.DeleteChatSessionResp{}, nil
}
