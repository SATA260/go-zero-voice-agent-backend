// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package voice

import (
	"context"

	"net/http"

	"go-zero-voice-agent/app/voicechat/cmd/api/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/types"
	
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/websocket"

	wsTool "github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type StartLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建websocket连接,然后帮助建立webrtc连接
func NewStartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartLogic {
	return &StartLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StartLogic) Start(req *types.StartVoiceRequest, r *http.Request, w http.ResponseWriter) (resp *types.Empty, err error) {
	conn, err := websocket.NewConnection(w, r, req.UserId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to establish websocket connection")
	}
	go l.handleWebsocketMsg(l.ctx, conn)

	return
}

func (l *StartLogic) handleWebsocketMsg(ctx context.Context, conn *wsTool.Conn) {
	
}
