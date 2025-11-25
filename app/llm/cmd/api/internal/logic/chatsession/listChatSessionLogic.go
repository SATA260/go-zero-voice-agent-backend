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

type ListChatSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 分页查询会话列表
func NewListChatSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListChatSessionLogic {
	return &ListChatSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListChatSessionLogic) ListChatSession(req *types.ListChatSessionReq) (resp *types.ListChatSessionResp, err error) {
	if req == nil {
		return nil, errors.New("invalid request")
	}

	rpcReq := toRpcListChatSessionReq(req)
	if rpcReq == nil {
		return nil, errors.New("invalid request")
	}

	rpcResp, err := l.svcCtx.ChatSessionRpc.ListChatSession(l.ctx, rpcReq)
	if err != nil {
		return nil, err
	}

	sessions := make([]types.ChatSession, 0, len(rpcResp.GetSessions()))
	for _, item := range rpcResp.GetSessions() {
		if item == nil {
			continue
		}
		if item.UserId != req.UserId {
			// Skip sessions not belonging to current user just in case RPC filter missed any.
			continue
		}
		sessions = append(sessions, toTypesChatSession(item))
	}

	return &types.ListChatSessionResp{
		Total:    rpcResp.GetTotal(),
		Sessions: sessions,
	}, nil
}
