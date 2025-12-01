// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package doc

import (
	"context"
	"mime/multipart"

	"go-zero-voice-agent/app/rag/cmd/api/internal/svc"
	"go-zero-voice-agent/app/rag/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UploadAndEmbedLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 上传文件并向量化
func NewUploadAndEmbedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadAndEmbedLogic {
	return &UploadAndEmbedLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UploadAndEmbedLogic) UploadAndEmbed(req *types.UploadDocReq, file multipart.File, header *multipart.FileHeader) (resp *types.UploadDocResp, err error) {
	

	return
}
