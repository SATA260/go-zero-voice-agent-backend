package result

import (
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-voice-agent/pkg/xerr"
	"google.golang.org/grpc/status"
	"net/http"
)

func HttpResult(r *http.Request, w http.ResponseWriter, resp interface{}, err error) {
	if err == nil {
		r := Success(resp)
		httpx.WriteJson(w, http.StatusOK, r)
		return
	}

	errCode := xerr.SERVER_COMMON_ERROR
	errMsg := "服务器开小差了，请稍后再试"

	// 获取错误的根本原因
	causeErr := errors.Cause(err)

	// 判断错误类型并设置相应的错误码和错误信息
	if e, ok := err.(*xerr.CodeError); ok {
		// 如果是自定义的CodeError类型，直接获取错误码和错误信息
		errCode = e.GetErrCode()
		errMsg = e.GetErrMsg()
	} else {
		// 如果不是CodeError类型，尝试从gRPC状态中获取错误信息
		if gstatus, ok := status.FromError(causeErr); ok {
			grpcCode := uint32(gstatus.Code())
			if xerr.IsCodeErr(grpcCode) {
				errCode = grpcCode
				errMsg = gstatus.Message()
			}
		}
	}

	logx.WithContext(r.Context()).Errorf("【API-ERR】 : %+v ", err)

	httpx.WriteJson(w, http.StatusBadRequest, Error(errCode, errMsg))
}
