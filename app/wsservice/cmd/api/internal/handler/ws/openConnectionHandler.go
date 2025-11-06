// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ws

import (
	"net/http"

	"go-zero-voice-agent/app/wsservice/cmd/api/internal/logic/ws"
	"go-zero-voice-agent/app/wsservice/cmd/api/internal/svc"
	"go-zero-voice-agent/pkg/tool"
)

// 创建 WebSocket 连接
func OpenConnectionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取用户id
		userID, err := tool.GetUserIdFromHeader(r)
		if err != nil {
			http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// 升级为 WebSocket 连接, 并注册到manager
		l := ws.NewOpenConnectionLogic(r.Context(), svcCtx)
		l.OpenConnection(w, r, userID)
	}
}
