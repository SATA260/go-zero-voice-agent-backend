// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ws

import (
	"context"
	"net/http"

	"go-zero-voice-agent/app/wsservice/cmd/api/internal/svc"
	"go-zero-voice-agent/app/wsservice/cmd/api/internal/websocket"

	"github.com/zeromicro/go-zero/core/logx"
)

type OpenConnectionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建 WebSocket 连接
func NewOpenConnectionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OpenConnectionLogic {
	return &OpenConnectionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OpenConnectionLogic) OpenConnection(w http.ResponseWriter, r *http.Request, userId string) {
	websocket.NewConnection(l.svcCtx.WsManager, w, r, userId)
	return
}
