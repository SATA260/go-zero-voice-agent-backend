// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package tts

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/logic/tts"
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/types"
)

// 获取TTS配置详情
func GetTtsConfigHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetTtsConfigReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := tts.NewGetTtsConfigLogic(r.Context(), svcCtx)
		resp, err := l.GetTtsConfig(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
