// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package chat

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
	conn, err := websocket.NewConnection(w, r)
	defer func ()  {
		err := conn.Close()
		if err != nil {
			l.Logger.Errorf("fail to close websocket connection")
		}
	} ()
	if err != nil {
		return nil, errors.Wrap(err, "failed to establish websocket connection")
	}
	l.handleWebsocketMsg(l.ctx, conn, req)

	return
}

func (l *StartLogic) handleWebsocketMsg(ctx context.Context, conn *wsTool.Conn, req *types.StartVoiceRequest) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var isClientCreated bool

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
				if isClientCreated {
					l.Logger.Info("Client already created, ignoring offer")
					continue
				}

				signalingClientParams := webrtc.SignalingClientParams{
					Ctx:               ctx,
					LlmService:        l.svcCtx.LlmChatServiceRpc,
					LlmConfig:         msg.LlmConfig,
					LlmConversationID: msg.LlmConversationID,
					SystemPrompt:      msg.SystemPrompt,
					UserID:            req.UserId,
					OutConn:           conn,
					ServerAddr:        l.svcCtx.Config.RustPBXConfig.WebSocketUrl,
					Initial: webrtc.PBXMessage{
						Command: webrtc.WS_CALLBACK_EVENT_TYPE_INVITE,
						Option: &webrtc.CallOptions{
							Asr: &webrtc.AsrConfig{
								Language:  msg.AsrConfig.Language,
								Provider:  msg.AsrConfig.Provider,
								AppId:     msg.AsrConfig.AppId,
								SecretId:  msg.AsrConfig.SecretId,
								SecretKey: msg.AsrConfig.SecretKey,
							},
							Tts: &webrtc.TtsConfig{
								Provider:  msg.TtsConfig.Provider,
								Speaker:   "603004",
								Speed:     1,
								Volume:    5,
								AppId:     msg.TtsConfig.AppId,
								SecretId:  msg.TtsConfig.SecretId,
								SecretKey: msg.TtsConfig.SecretKey,
							},
							Offer: msg.SDP,
						},
					},
				}

				client, err := webrtc.NewSignalingClient(signalingClientParams)
				if err != nil {
					l.Logger.Errorf("Failed to create signaling client: %v", err)
					cancel()
					continue
				}
				isClientCreated = true
				go client.Listen(msg.KnowledgeInfo)
				go client.HandleEvtMsg()
			}
		}
	}
}
