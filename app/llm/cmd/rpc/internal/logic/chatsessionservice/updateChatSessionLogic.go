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

type UpdateChatSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateChatSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateChatSessionLogic {
	return &UpdateChatSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateChatSessionLogic) UpdateChatSession(in *pb.UpdateChatSessionReq) (*pb.UpdateChatSessionResp, error) {
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

	convID := strings.TrimSpace(in.GetConvId())
	if convID == "" {
		convID = session.ConvId
	}
	session.ConvId = convID

	session.UserId = toNullInt64(in.GetUserId())
	session.Title = strings.TrimSpace(in.GetTitle())

	version := in.GetVersion()
	if version == 0 {
		version = session.Version
	}
	session.Version = version

	if err := l.svcCtx.ChatSessionModel.UpdateWithVersion(l.ctx, nil, session); err != nil {
		return nil, errors.Wrapf(err, "update chat session failed, id: %d", in.GetId())
	}

	return &pb.UpdateChatSessionResp{}, nil
}
