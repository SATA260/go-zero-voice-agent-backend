package docservicelogic

import (
	"context"
	"fmt"
	"io"
	"time"

	"go-zero-voice-agent/app/rag/cmd/rpc/internal/consts"
	"go-zero-voice-agent/app/rag/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/rag/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UploadFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUploadFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadFileLogic {
	return &UploadFileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UploadFileLogic) UploadFile(stream pb.DocService_UploadFileServer) error {
	firstChunk, err := stream.Recv()
	if err != nil {
		return err
	}

	userID := firstChunk.GetUserId()
	fileName := firstChunk.GetFileName()
	filePath := firstChunk.GetFilePath()
	contentType := firstChunk.GetContentType()
	fileSize := firstChunk.GetFileSize()

	if filePath == "" {
		return fmt.Errorf("file path must not be empty")
	}

	// Provide a size hint to MinIO when known; fall back to streaming uploads otherwise.
	if fileSize <= 0 {
		if len(firstChunk.GetChunk()) > 0 {
			fileSize = -1
		}
	}

	metadata := map[string]string{
		"user_id":     userID,
		"file_name":   fileName,
		"upload_time": time.Now().Format("2006-01-02 15:04:05"),
	}

	if err := l.svcCtx.MinioClient.EnsureBucket(l.ctx, consts.MINIO_BUCKETNAME_RAG_DOCUMENT); err != nil {
		return err
	}

	reader, writer := io.Pipe()
	writeErrCh := make(chan error, 1)

	go func() {
		defer close(writeErrCh)

		writeChunk := func(data []byte) error {
			for len(data) > 0 {
				n, err := writer.Write(data)
				if err != nil {
					return err
				}
				data = data[n:]
			}
			return nil
		}

		if len(firstChunk.GetChunk()) > 0 {
			if err := writeChunk(firstChunk.GetChunk()); err != nil {
				writeErrCh <- err
				_ = writer.CloseWithError(err)
				return
			}
		}

		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				writeErrCh <- err
				_ = writer.CloseWithError(err)
				return
			}

			if len(chunk.GetChunk()) == 0 {
				continue
			}

			if err := writeChunk(chunk.GetChunk()); err != nil {
				writeErrCh <- err
				_ = writer.CloseWithError(err)
				return
			}
		}

		if err := writer.Close(); err != nil {
			writeErrCh <- err
		}
	}()

	uploadInfo, err := l.svcCtx.MinioClient.Upload(
		l.ctx,
		consts.MINIO_BUCKETNAME_RAG_DOCUMENT,
		filePath,
		reader,
		fileSize,
		contentType,
		metadata,
	)
	if err != nil {
		_ = reader.CloseWithError(err)
		return err
	}

	if err := <-writeErrCh; err != nil {
		return err
	}

	objectKey := filePath
	if uploadInfo.Key != "" {
		objectKey = uploadInfo.Key
	}

	return stream.SendAndClose(&pb.UploadFileResp{FilePath: objectKey})
}
