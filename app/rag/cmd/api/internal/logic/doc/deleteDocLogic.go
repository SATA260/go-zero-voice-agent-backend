// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package doc

import (
	"context"
	"fmt"
	"strconv"

	"go-zero-voice-agent/app/rag/cmd/api/internal/svc"
	"go-zero-voice-agent/app/rag/cmd/api/internal/types"
	"go-zero-voice-agent/app/rag/cmd/rpc/client/docservice"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteDocLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除上传文件
func NewDeleteDocLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteDocLogic {
	return &DeleteDocLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteDocLogic) DeleteDoc(req *types.DeleteDocReq) (resp *types.DeleteDocResp, err error) {
	if req == nil {
		return nil, fmt.Errorf("request must not be nil")
	}

	if req.UserId <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}

	if req.Id <= 0 {
		return nil, fmt.Errorf("invalid document id")
	}

	rpcReq := &docservice.DeleteDocumentsReq{
		UserId: strconv.FormatInt(req.UserId, 10),
		Ids:    []string{strconv.FormatInt(req.Id, 10)},
	}

	rpcResp, err := l.svcCtx.DocService.DeleteDocuments(l.ctx, rpcReq)
	if err != nil {
		l.Logger.Errorf("delete document rpc failed: %v", err)
		return nil, err
	}

	var deleted int32
	if rpcResp != nil {
		deleted = rpcResp.GetDeletedCount()
	}

	return &types.DeleteDocResp{DeletedCount: deleted}, nil
}
