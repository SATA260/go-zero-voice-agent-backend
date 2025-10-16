// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.1

package user

import (
	"go-zero-voice-agent/pkg/result"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-voice-agent/app/usercenter/cmd/api/internal/logic/user"
	"go-zero-voice-agent/app/usercenter/cmd/api/internal/svc"
	"go-zero-voice-agent/app/usercenter/cmd/api/internal/types"
)

// wechat mini auth
func WxMiniAuthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.WXMiniAuthReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := user.NewWxMiniAuthLogic(r.Context(), svcCtx)
		resp, err := l.WxMiniAuth(&req)
		result.HttpResult(r, w, resp, err)
	}
}
