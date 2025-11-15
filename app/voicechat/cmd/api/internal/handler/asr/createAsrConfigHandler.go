// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package asr

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/logic/asr"
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/api/internal/types"
)

// 创建ASR配置
func CreateAsrConfigHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateAsrConfigReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := asr.NewCreateAsrConfigLogic(r.Context(), svcCtx)
		resp, err := l.CreateAsrConfig(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
