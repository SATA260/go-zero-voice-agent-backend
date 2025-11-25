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

type CreateChatSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateChatSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateChatSessionLogic {
	return &CreateChatSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateChatSessionLogic) CreateChatSession(in *pb.CreateChatSessionReq) (*pb.CreateChatSessionResp, error) {
	if in == nil {
		return nil, errors.New("invalid request")
	}

	session := &model.ChatSession{
		ConvId: normalizeConvID(in.GetConvId()),
		UserId: toNullInt64(in.GetUserId()),
		Title:  strings.TrimSpace(in.GetTitle()),
	}

	result, err := l.svcCtx.ChatSessionModel.Insert(l.ctx, nil, session)
	if err != nil {
		return nil, errors.Wrapf(err, "create chat session failed, req: %+v", in)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "fetch last insert id failed")
	}

	session.Id = id

	return &pb.CreateChatSessionResp{Id: id}, nil
}
