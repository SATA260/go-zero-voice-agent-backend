// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-voice-agent/app/llm/cmd/api/internal/logic/config"
	"go-zero-voice-agent/app/llm/cmd/api/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/api/internal/types"
)

// 分页查询我的配置
func ListMyConfigHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListMyConfigReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := config.NewListMyConfigLogic(r.Context(), svcCtx)
		resp, err := l.ListMyConfig(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
