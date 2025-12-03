package docservicelogic

import (
	"context"
	"strings"

	"go-zero-voice-agent/app/rag/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/rag/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ListDocumentsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListDocumentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDocumentsLogic {
	return &ListDocumentsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListDocumentsLogic) ListDocuments(in *pb.ListDocumentsReq) (*pb.ListDocumentsResp, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request must not be nil")
	}

	builder := l.svcCtx.FileUploadModel.SelectBuilder()

	userID := in.GetUserId()
	if userID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	builder = builder.Where("user_id = ?", userID)

	if filter := in.GetFilter(); filter != nil {
		if name := strings.TrimSpace(filter.GetFileName()); name != "" {
			builder = builder.Where("file_name LIKE ?", "%"+name+"%")
		}

		if format := strings.TrimSpace(filter.GetFileFormat()); format != "" {
			builder = builder.Where("LOWER(file_format) = ?", strings.ToLower(format))
		}
	}

	page := int64(1)
	pageSize := int64(20)
	orderBy := ""

	if pq := in.GetPageQuery(); pq != nil {
		if pq.Page > 0 {
			page = pq.Page
		}
		if pq.PageSize > 0 {
			pageSize = pq.PageSize
		}
		if order := strings.TrimSpace(pq.GetOrderBy()); order != "" {
			if normalized := normalizeOrderBy(order); normalized != "" {
				orderBy = normalized
			} else {
				return nil, status.Error(codes.InvalidArgument, "unsupported order_by value")
			}
		}
	}

	if pageSize > 200 {
		pageSize = 200
	}

	records, err := l.svcCtx.FileUploadModel.FindPageListByPage(l.ctx, builder, page, pageSize, orderBy)
	if err != nil {
		l.Logger.Errorf("list documents failed: %v", err)
		return nil, status.Error(codes.Internal, "query documents failed")
	}

	if len(records) == 0 {
		return &pb.ListDocumentsResp{}, nil
	}

	results := make([]*pb.ListDocumentsItem, 0)

	for _, record := range records {
		result := &pb.ListDocumentsItem{
			Id:       record.Id,
			FileName: record.FileName,
			Status: record.Status,
		}

		if record.FileFormat.Valid {
			result.FileFormat = record.FileFormat.String
		}
		
		results = append(results, result)
	}

	return &pb.ListDocumentsResp{
		Results: results,
	}, nil
}

func normalizeOrderBy(order string) string {
	lower := strings.ToLower(order)
	switch lower {
	case "id desc":
		return "id DESC"
	case "id asc":
		return "id ASC"
	case "create_time desc":
		return "create_time DESC"
	case "create_time asc":
		return "create_time ASC"
	default:
		return ""
	}
}
