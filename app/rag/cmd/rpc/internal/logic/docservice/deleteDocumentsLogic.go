package docservicelogic

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"go-zero-voice-agent/app/rag/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/rag/cmd/rpc/pb"

	"github.com/Masterminds/squirrel"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DeleteDocumentsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteDocumentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteDocumentsLogic {
	return &DeleteDocumentsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteDocumentsLogic) DeleteDocuments(in *pb.DeleteDocumentsReq) (*pb.DeleteDocumentsResp, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request must not be nil")
	}

	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	cleanedIDs := make([]string, 0, len(in.GetIds()))
	for _, id := range in.GetIds() {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			cleanedIDs = append(cleanedIDs, trimmed)
		}
	}

	if len(cleanedIDs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "ids must not be empty")
	}

	resp, err := l.svcCtx.RagClient.DeleteDocuments(l.ctx, userID, cleanedIDs)
	if err != nil {
		l.Logger.Errorf("delete rag documents failed: %v", err)
		return nil, status.Error(codes.Internal, "rag service delete documents failed")
	}

	if err := l.removeFileUploadRecords(cleanedIDs); err != nil {
		l.Logger.Errorf("delete file upload records failed: %v", err)
		return nil, status.Error(codes.Internal, "delete document records failed")
	}

	out := &pb.DeleteDocumentsResp{}
	if resp != nil && resp.DeletedCount > 0 {
		out.DeletedCount = int32(resp.DeletedCount)
	}

	return out, nil
}

// removeFileUploadRecords deletes matching file_upload rows for the provided identifiers.
func (l *DeleteDocumentsLogic) removeFileUploadRecords(ids []string) error {
	resolvedIDs, err := l.resolveFileUploadRecordIDs(ids)
	if err != nil {
		return err
	}

	if len(resolvedIDs) == 0 {
		return nil
	}

	return l.svcCtx.FileUploadModel.Trans(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		for _, id := range resolvedIDs {
			if err := l.svcCtx.FileUploadModel.Delete(ctx, session, id); err != nil {
				return err
			}
		}
		return nil
	})
}

func (l *DeleteDocumentsLogic) resolveFileUploadRecordIDs(ids []string) ([]int64, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	uniqueIDs := make(map[int64]struct{}, len(ids))
	uniquePaths := make(map[string]struct{})
	paths := make([]string, 0)

	for _, rawID := range ids {
		if parsed, err := strconv.ParseInt(rawID, 10, 64); err == nil && parsed > 0 {
			uniqueIDs[parsed] = struct{}{}
			continue
		}

		if _, exists := uniquePaths[rawID]; exists {
			continue
		}
		uniquePaths[rawID] = struct{}{}
		paths = append(paths, rawID)
	}

	if len(paths) > 0 {
		builder := l.svcCtx.FileUploadModel.SelectBuilder().Where(squirrel.Eq{"file_path": paths})
		records, err := l.svcCtx.FileUploadModel.FindAll(l.ctx, builder, "")
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			if record == nil {
				continue
			}
			if record.Id > 0 {
				uniqueIDs[record.Id] = struct{}{}
			}
		}
	}

	if len(uniqueIDs) == 0 {
		return nil, nil
	}

	resolved := make([]int64, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		resolved = append(resolved, id)
	}
	sort.Slice(resolved, func(i, j int) bool { return resolved[i] < resolved[j] })
	return resolved, nil
}
