package chatmessageservicelogic

import (
	"context"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteChatMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteChatMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteChatMessageLogic {
	return &DeleteChatMessageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteChatMessageLogic) DeleteChatMessage(in *pb.DeleteChatMessageReq) (*pb.DeleteChatMessageResp, error) {
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

	if version := in.GetVersion(); version > 0 {
		message.Version = version
	}

	if err := l.svcCtx.ChatMessageModel.DeleteSoft(l.ctx, nil, message); err != nil {
		return nil, errors.Wrapf(err, "delete chat message failed, id: %d", in.GetId())
	}

	return &pb.DeleteChatMessageResp{}, nil
}
