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

type CreateChatMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateChatMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateChatMessageLogic {
	return &CreateChatMessageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateChatMessageLogic) CreateChatMessage(in *pb.CreateChatMessageReq) (*pb.CreateChatMessageResp, error) {
	if in == nil {
		return nil, errors.New("invalid request")
	}

	if in.GetSessionId() <= 0 {
		return nil, errors.New("session_id must be greater than 0")
	}

	role := strings.TrimSpace(in.GetRole())
	if role == "" {
		return nil, errors.New("role is required")
	}

	message := &model.ChatMessage{
		SessionId:  in.GetSessionId(),
		Role:       role,
		Content:    toNullString(in.GetContent()),
		Extra:      toNullString(in.GetExtra()),
		ToolCalls:  toolCallsToModel(in.GetToolCalls()),
		ToolCallId: toNullString(in.GetToolCallId()),
	}

	result, err := l.svcCtx.ChatMessageModel.Insert(l.ctx, nil, message)
	if err != nil {
		return nil, errors.Wrapf(err, "create chat message failed, req: %+v", in)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "fetch last insert id failed")
	}

	message.Id = id

	return &pb.CreateChatMessageResp{Id: id}, nil
}
