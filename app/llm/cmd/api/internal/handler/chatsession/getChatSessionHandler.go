// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package chatsession

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-voice-agent/app/llm/cmd/api/internal/logic/chatsession"
	"go-zero-voice-agent/app/llm/cmd/api/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/api/internal/types"
)

// 查询会话详情
func GetChatSessionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetChatSessionReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := chatsession.NewGetChatSessionLogic(r.Context(), svcCtx)
		resp, err := l.GetChatSession(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
