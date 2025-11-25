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

type DeleteChatSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除会话
func NewDeleteChatSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteChatSessionLogic {
	return &DeleteChatSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteChatSessionLogic) DeleteChatSession(req *types.DeleteChatSessionReq) (resp *types.DeleteChatSessionResp, err error) {
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
		return nil, errors.New("not authorized to delete this session")
	}

	_, err = l.svcCtx.ChatSessionRpc.DeleteChatSession(l.ctx, toRpcDeleteChatSessionReq(req.Id))
	if err != nil {
		return nil, err
	}

	return &types.DeleteChatSessionResp{}, nil
}
