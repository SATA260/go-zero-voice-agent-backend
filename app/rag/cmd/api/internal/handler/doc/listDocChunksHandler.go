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

// 分页查询文件切片
func ListDocChunksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListDocChunksReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := doc.NewListDocChunksLogic(r.Context(), svcCtx)
		resp, err := l.ListDocChunks(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
