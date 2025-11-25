// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package chatmessage

import (
	"context"

	"go-zero-voice-agent/app/llm/cmd/api/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/api/internal/types"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/chatsessionservice"
	"go-zero-voice-agent/app/llm/pkg/consts"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListChatMessageBySessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 根据会话分页查询消息
func NewListChatMessageBySessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListChatMessageBySessionLogic {
	return &ListChatMessageBySessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListChatMessageBySessionLogic) ListChatMessageBySession(req *types.ListChatMessageBySessionReq) (resp *types.ListChatMessageBySessionResp, err error) {
	if req == nil {
		return nil, errors.New("invalid request")
	}

	if req.SessionId <= 0 {
		return nil, errors.New("sessionId must be greater than 0")
	}

	sessionResp, err := l.svcCtx.ChatSessionRpc.GetChatSession(l.ctx, &chatsessionservice.GetChatSessionReq{Id: req.SessionId})
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

	rpcReq := toRpcListChatMessageReq(req)
	if rpcReq == nil {
		return nil, errors.New("invalid request")
	}

	rpcResp, err := l.svcCtx.ChatMessageRpc.ListChatMessage(l.ctx, rpcReq)
	if err != nil {
		return nil, err
	}

	messages := make([]types.ChatMessage, 0, len(rpcResp.GetMessages()))
	for _, msg := range rpcResp.GetMessages() {
		if msg == nil {
			continue
		}
		messages = append(messages, toTypesChatMessage(msg))
	}

	if messages[0].Role == consts.ChatMessageRoleSystem {
		messages = messages[1:]
	}

	return &types.ListChatMessageBySessionResp{
		Total:    rpcResp.GetTotal(),
		Messages: messages,
	}, nil
}
