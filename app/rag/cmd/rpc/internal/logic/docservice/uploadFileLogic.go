package docservicelogic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go-zero-voice-agent/app/rag/cmd/rpc/internal/consts"
	"go-zero-voice-agent/app/rag/cmd/rpc/internal/ragclient"
	"go-zero-voice-agent/app/rag/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/rag/cmd/rpc/pb"
	"go-zero-voice-agent/app/rag/model"

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

	if userID <= 0 {
		return errors.New("user id is required")
	}

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
		"user_id":     strconv.FormatInt(userID, 10),
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

	var userIDValue sql.NullInt64
	userIDValue = sql.NullInt64{Int64: userID, Valid: true}

	var fileFormat sql.NullString
	if ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), "."); ext != "" {
		fileFormat = sql.NullString{String: ext, Valid: true}
	}

	bucketName := sql.NullString{String: consts.MINIO_BUCKETNAME_RAG_DOCUMENT, Valid: true}

	record := &model.FileUpload{
		Version:    1,
		UserId:     userIDValue,
		BucketName: bucketName,
		FileName:   fileName,
		FileFormat: fileFormat,
		FilePath:   objectKey,
		StoreType:  consts.STORE_TYPE_MINIO,
		Status:     0,
	}

	insertResult, err := l.svcCtx.FileUploadModel.Insert(l.ctx, nil, record)
	if err != nil {
		l.Logger.Errorf("insert file upload record failed: %v", err)
		if rmErr := l.svcCtx.MinioClient.Remove(l.ctx, consts.MINIO_BUCKETNAME_RAG_DOCUMENT, objectKey); rmErr != nil {
			l.Logger.Errorf("rollback minio object failed: %v", rmErr)
		}
		return err
	}

	insertID, err := insertResult.LastInsertId()
	if err != nil {
		l.Logger.Errorf("fetch last insert id failed: %v", err)
	} else if insertID > 0 {
		record.Id = insertID
	}

	if err := stream.SendAndClose(&pb.UploadFileResp{FilePath: objectKey}); err != nil {
		return err
	}

	if userID <= 0 {
		l.Logger.Infof("skip embed: empty user id for file %s", objectKey)
		return nil
	}

	recordCopy := *record
	fileID := objectKey
	if recordCopy.Id > 0 {
		fileID = strconv.FormatInt(recordCopy.Id, 10)
	}

	embedReq := &ragclient.EmbedRequest{
		FileID:      fileID,
		BucketName:  consts.MINIO_BUCKETNAME_RAG_DOCUMENT,
		ObjectPath:  objectKey,
		Filename:    recordCopy.FileName,
		ContentType: contentType,
	}

	go func(rec model.FileUpload, req *ragclient.EmbedRequest) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := l.embedWithRetry(ctx, strconv.FormatInt(userID, 10), &rec, req); err != nil {
			l.Logger.Errorf("async embed failed for file %s: %v", req.ObjectPath, err)
		}
	}(recordCopy, embedReq)

	return nil
}

func (l *UploadFileLogic) embedWithRetry(ctx context.Context, userID string, record *model.FileUpload, embedReq *ragclient.EmbedRequest) error {
	const (
		maxAttempts       = 3
		initialBackoff    = 500 * time.Millisecond
		backoffMultiplier = 2
	)

	var lastErr error
	backoff := initialBackoff

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if _, err := l.svcCtx.RagClient.Embed(ctx, userID, embedReq); err != nil {
			lastErr = err
			l.Logger.Errorf("embed attempt %d failed: %v", attempt, err)
			if attempt == maxAttempts {
				break
			}

			timer := time.NewTimer(backoff)
			select {
			case <-timer.C:
				backoff *= backoffMultiplier
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			}
			continue
		}

		lastErr = nil

		if record.Id > 0 {
			record.Status = 1
			if _, err := l.svcCtx.FileUploadModel.Update(ctx, nil, record); err != nil {
				l.Logger.Errorf("update file status failed: %v", err)
			}
		}

		return nil
	}

	return fmt.Errorf("embed failed after retries: %w", lastErr)
}
