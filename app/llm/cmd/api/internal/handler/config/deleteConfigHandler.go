// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"net/http"

	"go-zero-voice-agent/app/llm/cmd/api/internal/logic/config"
	"go-zero-voice-agent/app/llm/cmd/api/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/api/internal/types"
	"go-zero-voice-agent/pkg/tool"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 删除配置
func DeleteConfigHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeleteConfigReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		userId, err := tool.GetUserIdInt64FromHeader(r)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		req.UserId = userId

		l := config.NewDeleteConfigLogic(r.Context(), svcCtx)
		resp, err := l.DeleteConfig(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
