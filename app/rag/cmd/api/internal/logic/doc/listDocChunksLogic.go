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

type ListDocChunksLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 分页查询文件切片
func NewListDocChunksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDocChunksLogic {
	return &ListDocChunksLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListDocChunksLogic) ListDocChunks(req *types.ListDocChunksReq) (resp *types.ListDocChunksResp, err error) {
	if req == nil {
		return nil, fmt.Errorf("request must not be nil")
	}

	if req.UserId <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}

	fileId := strings.TrimSpace(req.FileId)
	if fileId == "" {
		return nil, fmt.Errorf("fileId must not be empty")
	}

	rpcReq := &docservice.ListChunksReq{
		UserId: req.UserId,
		FileId: fileId,
	}

	trimmedOrder := strings.TrimSpace(req.PageQuery.OrderBy)
	if req.PageQuery.Page > 0 || req.PageQuery.PageSize > 0 || trimmedOrder != "" {
		rpcReq.PageQuery = &docservice.PageQuery{
			Page:     req.PageQuery.Page,
			PageSize: req.PageQuery.PageSize,
			OrderBy:  trimmedOrder,
		}
	}

	rpcResp, err := l.svcCtx.DocService.ListChunks(l.ctx, rpcReq)
	if err != nil {
		l.Logger.Errorf("list doc chunks rpc failed: %v", err)
		return nil, err
	}

	out := &types.ListDocChunksResp{
		ChunkList: make([]types.DocChunkItem, 0),
	}

	if rpcResp == nil {
		return out, nil
	}

	chunks := rpcResp.GetChunks()
	if len(chunks) > 0 {
		out.ChunkList = make([]types.DocChunkItem, 0, len(chunks))
		for _, record := range chunks {
			if record == nil {
				continue
			}

			metadata := record.GetMetadata()
			mdCopy := make(map[string]string, len(metadata))
			for k, v := range metadata {
				mdCopy[k] = v
			}

			out.ChunkList = append(out.ChunkList, types.DocChunkItem{
				CustomId:    record.GetCustomId(),
				PageContent: record.GetPageContent(),
				Metadata:    mdCopy,
			})
		}
	}

	out.Total = rpcResp.GetTotal()

	return out, nil
}
