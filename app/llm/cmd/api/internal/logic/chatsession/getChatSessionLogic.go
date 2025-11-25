// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package chatsession

import (
	"context"

	"go-zero-voice-agent/app/llm/cmd/api/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/api/internal/types"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetChatSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询会话详情
func NewGetChatSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetChatSessionLogic {
	return &GetChatSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetChatSessionLogic) GetChatSession(req *types.GetChatSessionReq) (resp *types.GetChatSessionResp, err error) {
	if req == nil {
		return nil, errors.New("invalid request")
	}

	sessionResp, err := l.svcCtx.ChatSessionRpc.GetChatSession(l.ctx, toRpcGetChatSessionReq(req.Id))
	if err != nil {
		return nil, err
	}

	session := sessionResp.GetSession()
	if session == nil {
		return nil, errors.New("session not found")
	}

	if session.UserId != req.UserId {
		return nil, errors.New("not authorized to access this session")
	}

	return &types.GetChatSessionResp{Session: toTypesChatSession(session)}, nil
}
