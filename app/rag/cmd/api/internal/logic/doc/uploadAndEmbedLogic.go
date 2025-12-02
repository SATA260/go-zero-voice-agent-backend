// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package doc

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"go-zero-voice-agent/app/rag/cmd/api/internal/svc"
	"go-zero-voice-agent/app/rag/cmd/api/internal/types"
	"go-zero-voice-agent/app/rag/cmd/rpc/client/docservice"

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
	stream, err := l.svcCtx.DocService.UploadFile(l.ctx)
	if err != nil {
		return nil, fmt.Errorf("open upload stream: %w", err)
	}

	if req.UserId <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}

	filename := header.Filename
	if req.FileName != "" {
		filename = req.FileName
	}
	filename = filepath.Base(filename)

	userIDStr := strconv.FormatInt(req.UserId, 10)
	objectKey := fmt.Sprintf("rag/doc/user_%s/%d_%s", userIDStr, time.Now().UnixNano(), filename)

	sizeHint := header.Size
	contentType := header.Header.Get("Content-Type")

	const chunkSize = 512 * 1024 // 512KB per chunk keeps memory modest while utilising bandwidth.
	buffer := make([]byte, chunkSize)

	var readErr error
	var n int

	if n, readErr = file.Read(buffer); readErr != nil && readErr != io.EOF {
		return nil, fmt.Errorf("read file: %w", readErr)
	}

	if contentType == "" && n > 0 {
		sniffLen := n
		if sniffLen > 512 {
			sniffLen = 512
		}
		contentType = http.DetectContentType(buffer[:sniffLen])
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	firstChunk := &docservice.UploadFileReq{
		UserId:      userIDStr,
		FileName:    filename,
		FilePath:    objectKey,
		ContentType: contentType,
		FileSize:    sizeHint,
	}
	if n > 0 {
		firstChunk.Chunk = append([]byte(nil), buffer[:n]...)
	}

	if err := stream.Send(firstChunk); err != nil {
		return nil, fmt.Errorf("send first chunk: %w", err)
	}

	for readErr != io.EOF {
		n, readErr = file.Read(buffer)
		if n > 0 {
			chunk := &docservice.UploadFileReq{Chunk: append([]byte(nil), buffer[:n]...)}
			if err := stream.Send(chunk); err != nil {
				return nil, fmt.Errorf("send chunk: %w", err)
			}
		}
		if readErr != nil && readErr != io.EOF {
			return nil, fmt.Errorf("read file: %w", readErr)
		}
	}

	rpcResp, err := stream.CloseAndRecv()
	if err != nil {
		return nil, fmt.Errorf("close upload stream: %w", err)
	}

	return &types.UploadDocResp{Path: rpcResp.GetFilePath()}, nil
}
