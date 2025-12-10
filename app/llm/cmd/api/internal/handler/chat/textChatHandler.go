// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go-zero-voice-agent/app/llm/cmd/api/internal/logic/chat"
	"go-zero-voice-agent/app/llm/cmd/api/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 进行文字对话
func TextChatHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.TextChatReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 非流式响应，直接http返回
		if !req.IsStream {
			l := chat.NewTextChatLogic(r.Context(), svcCtx)
			resp, err := l.TextChat(&req)
			if err != nil {
				httpx.ErrorCtx(r.Context(), w, err)
			} else {
				httpx.OkJsonCtx(r.Context(), w, resp)
			}
			return
		}

		// 流式响应，使用SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := w.(http.Flusher)
		if !ok {
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("streaming unsupported"))
			return
		}

		l := chat.NewTextChatLogic(r.Context(), svcCtx)
		rpcStream, err := l.TextChatStream(&req)
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
			flusher.Flush()
			return
		}

		for {
			resp, err := rpcStream.Recv()
			if err == io.EOF {
				break
			}

			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
				flusher.Flush()
				break
			}

			dataBytes, err := json.Marshal(resp)
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
				flusher.Flush()
				break
			}

			fmt.Fprintf(w, "data: %s\n\n", dataBytes)
			flusher.Flush()
		}

	}
}
