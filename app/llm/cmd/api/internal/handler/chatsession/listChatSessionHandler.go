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

// 分页查询会话列表
func ListChatSessionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListChatSessionReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := chatsession.NewListChatSessionLogic(r.Context(), svcCtx)
		resp, err := l.ListChatSession(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
