// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package doc

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-voice-agent/app/rag/cmd/api/internal/logic/doc"
	"go-zero-voice-agent/app/rag/cmd/api/internal/svc"
	"go-zero-voice-agent/app/rag/cmd/api/internal/types"
)

// 分页查询上传文件
func ListDocHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListDocReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := doc.NewListDocLogic(r.Context(), svcCtx)
		resp, err := l.ListDoc(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
