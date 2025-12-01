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

// 上传文件并向量化
func UploadAndEmbedHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UploadDocReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		defer file.Close()

		l := doc.NewUploadAndEmbedLogic(r.Context(), svcCtx)
		resp, err := l.UploadAndEmbed(&req, file, header)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
