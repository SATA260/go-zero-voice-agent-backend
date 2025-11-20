// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package chat

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/logic/chat"
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/types"
)

// 创建websocket连接,然后帮助建立webrtc连接
func StartHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.StartVoiceRequest
		// if err := httpx.Parse(r, &req); err != nil {
		// 	httpx.ErrorCtx(r.Context(), w, err)
		// 	return
		// }
		req.UserId = 1

		l := chat.NewStartLogic(r.Context(), svcCtx)
		_, err := l.Start(&req, r, w)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		}
	}
}

