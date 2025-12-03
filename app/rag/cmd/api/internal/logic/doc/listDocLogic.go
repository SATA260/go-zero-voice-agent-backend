// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package doc

import (
	"context"
	"fmt"
	"strings"

	"go-zero-voice-agent/app/rag/cmd/api/internal/svc"
	"go-zero-voice-agent/app/rag/cmd/api/internal/types"
	"go-zero-voice-agent/app/rag/cmd/rpc/client/docservice"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListDocLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 分页查询上传文件
func NewListDocLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDocLogic {
	return &ListDocLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListDocLogic) ListDoc(req *types.ListDocReq) (resp *types.ListDocResp, err error) {
	if req == nil {
		return nil, fmt.Errorf("request must not be nil")
	}

	if req.UserId <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}

	rpcReq := &docservice.ListDocumentsReq{
		UserId: req.UserId,
	}

	trimmedOrder := strings.TrimSpace(req.PageQuery.OrderBy)
	if req.PageQuery.Page > 0 || req.PageQuery.PageSize > 0 || trimmedOrder != "" {
		rpcReq.PageQuery = &docservice.PageQuery{
			Page:     req.PageQuery.Page,
			PageSize: req.PageQuery.PageSize,
			OrderBy:  trimmedOrder,
		}
	}

	trimmedName := strings.TrimSpace(req.Filter.FileName)
	trimmedFormat := strings.TrimSpace(req.Filter.FileFormat)
	if trimmedName != "" || trimmedFormat != "" {
		rpcReq.Filter = &docservice.ListDocumentsFilter{
			FileName:   trimmedName,
			FileFormat: trimmedFormat,
		}
	}

	rpcResp, err := l.svcCtx.DocService.ListDocuments(l.ctx, rpcReq)
	if err != nil {
		l.Logger.Errorf("list documents rpc failed: %v", err)
		return nil, err
	}

	out := &types.ListDocResp{
		DocumentList: make([]types.DocumentItem, 0),
	}

	if rpcResp == nil {
		return out, nil
	}

	results := rpcResp.GetResults()
	if len(results) > 0 {
		out.DocumentList = make([]types.DocumentItem, 0, len(results))
		for _, item := range results {
			if item == nil {
				continue
			}

			out.DocumentList = append(out.DocumentList, types.DocumentItem{
				Id:         item.GetId(),
				FileName:   item.GetFileName(),
				FileFormat: item.GetFileFormat(),
				Status:     item.GetStatus(),
			})
		}
	}

	out.Total = int64(len(out.DocumentList))

	return out, nil
}
