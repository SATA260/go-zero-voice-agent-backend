// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package voice

import (
	"context"
	"encoding/json"
	"net/http"

	"go-zero-voice-agent/app/voicechat/cmd/api/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/types"
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/webrtc"
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
	for {
		select {
		case <-ctx.Done():
			return
		default:
			typeVal, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if typeVal != wsTool.TextMessage {
				continue
			}
			l.Logger.Infof("Received message: %s", string(data))
			var msg webrtc.WebRTCMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				l.Logger.Errorf("Failed to unmarshal message: %v", err)
				continue
			}
			if msg.Type == webrtc.WEBRTC_SIGNALING_OFFER {
				webrtc.NewSignalingClient(conn, ctx, l.svcCtx.Config.RustPBXConfig.WebSocketUrl, webrtc.PBXMessage{
					Command: webrtc.WS_CALLBACK_EVENT_TYPE_INVITE,
					Option: &webrtc.CallOptions{
						
					},
				})
			}
		}
	}
}
