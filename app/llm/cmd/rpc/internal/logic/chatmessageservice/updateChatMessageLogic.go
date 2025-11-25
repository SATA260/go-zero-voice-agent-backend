package chatmessageservicelogic

import (
	"context"
	"strings"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateChatMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateChatMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateChatMessageLogic {
	return &UpdateChatMessageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateChatMessageLogic) UpdateChatMessage(in *pb.UpdateChatMessageReq) (*pb.UpdateChatMessageResp, error) {
	if in == nil {
		return nil, errors.New("invalid request")
	}

	message, err := l.svcCtx.ChatMessageModel.FindOne(l.ctx, in.GetId())
	if err != nil {
		if err == model.ErrNotFound {
			return nil, errors.Wrapf(model.ErrNotFound, "chat message not found, id: %d", in.GetId())
		}
		return nil, errors.Wrapf(err, "find chat message failed, id: %d", in.GetId())
	}

	if sessionID := in.GetSessionId(); sessionID > 0 {
		message.SessionId = sessionID
	}

	message.ConfigId = toNullInt64(in.GetConfigId())

	if role := strings.TrimSpace(in.GetRole()); role != "" {
		message.Role = role
	}

	message.Content = toNullString(in.GetContent())
	message.Extra = toNullString(in.GetExtra())

	version := in.GetVersion()
	if version == 0 {
		version = message.Version
	}
	message.Version = version

	if err := l.svcCtx.ChatMessageModel.UpdateWithVersion(l.ctx, nil, message); err != nil {
		return nil, errors.Wrapf(err, "update chat message failed, id: %d", in.GetId())
	}

	return &pb.UpdateChatMessageResp{}, nil
}
