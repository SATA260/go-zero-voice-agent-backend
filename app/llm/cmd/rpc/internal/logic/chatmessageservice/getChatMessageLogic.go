package chatmessageservicelogic

import (
	"context"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetChatMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetChatMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetChatMessageLogic {
	return &GetChatMessageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetChatMessageLogic) GetChatMessage(in *pb.GetChatMessageReq) (*pb.GetChatMessageResp, error) {
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

	return &pb.GetChatMessageResp{Message: chatMessageToPb(message)}, nil
}
