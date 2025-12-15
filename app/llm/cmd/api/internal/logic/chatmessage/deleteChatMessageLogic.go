// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package chatmessage

import (
	"context"

	"go-zero-voice-agent/app/llm/cmd/api/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/api/internal/types"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/chatsessionservice"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteChatMessageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除聊天消息
func NewDeleteChatMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteChatMessageLogic {
	return &DeleteChatMessageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteChatMessageLogic) DeleteChatMessage(req *types.DeleteChatMessageReq) (resp *types.DeleteChatMessageResp, err error) {
	if req == nil {
		return nil, errors.New("invalid request")
	}

	messageResp, err := l.svcCtx.ChatMessageRpc.GetChatMessage(l.ctx, toRpcGetChatMessageReq(req.Id))
	if err != nil {
		return nil, err
	}

	message := messageResp.GetMessage()
	if message == nil {
		return nil, errors.New("message not found")
	}

	sessionResp, err := l.svcCtx.ChatSessionRpc.GetChatSession(l.ctx, &chatsessionservice.GetChatSessionReq{Id: message.SessionId})
	if err != nil {
		return nil, err
	}

	session := sessionResp.GetSession()
	if session == nil {
		return nil, errors.New("session not found")
	}

	if session.UserId != req.UserId {
		return nil, errors.New("not authorized to delete this message")
	}

	_, err = l.svcCtx.ChatMessageRpc.DeleteChatMessage(l.ctx, toRpcDeleteChatMessageReq(req.Id))
	if err != nil {
		return nil, err
	}

	return &types.DeleteChatMessageResp{}, nil
}
