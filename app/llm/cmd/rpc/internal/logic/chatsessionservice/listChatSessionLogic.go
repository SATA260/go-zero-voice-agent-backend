package chatsessionservicelogic

import (
	"context"
	"strings"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"

	"github.com/Masterminds/squirrel"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListChatSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListChatSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListChatSessionLogic {
	return &ListChatSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListChatSessionLogic) ListChatSession(in *pb.ListChatSessionReq) (*pb.ListChatSessionResp, error) {
	const (
		defaultPage     int64 = 1
		defaultPageSize int64 = 10
		maxPageSize     int64 = 100
	)

	page := defaultPage
	pageSize := defaultPageSize
	orderBy := ""

	if pq := in.GetPageQuery(); pq != nil {
		if pq.GetPage() > 0 {
			page = pq.GetPage()
		}
		if pq.GetPageSize() > 0 {
			pageSize = pq.GetPageSize()
		}
		if pageSize > maxPageSize {
			pageSize = maxPageSize
		}
		orderBy = sanitizeSessionOrderBy(pq.GetOrderBy())
	}

	builder := l.svcCtx.ChatSessionModel.SelectBuilder()

	if filter := in.GetFilter(); filter != nil {
		if filter.GetId() > 0 {
			builder = builder.Where(squirrel.Eq{"id": filter.GetId()})
		}
		if convID := strings.TrimSpace(filter.GetConvId()); convID != "" {
			builder = builder.Where("conv_id LIKE ?", "%"+convID+"%")
		}
		if filter.GetUserId() > 0 {
			builder = builder.Where(squirrel.Eq{"user_id": filter.GetUserId()})
		}
		if title := strings.TrimSpace(filter.GetTitle()); title != "" {
			builder = builder.Where("title LIKE ?", "%"+title+"%")
		}
	}

	records, total, err := l.svcCtx.ChatSessionModel.FindPageListByPageWithTotal(l.ctx, builder, page, pageSize, orderBy)
	if err != nil {
		return nil, errors.Wrapf(err, "list chat sessions failed, req: %+v", in)
	}

	sessions := make([]*pb.ChatSession, 0, len(records))
	for _, record := range records {
		sessions = append(sessions, chatSessionToPb(record))
	}

	return &pb.ListChatSessionResp{
		Total:    total,
		Sessions: sessions,
	}, nil
}

func sanitizeSessionOrderBy(orderBy string) string {
	orderBy = strings.TrimSpace(orderBy)
	if orderBy == "" {
		return ""
	}

	fields := map[string]string{
		"id":          "id",
		"create_time": "create_time",
		"update_time": "update_time",
		"user_id":     "user_id",
	}

	parts := strings.Fields(orderBy)
	if len(parts) == 0 {
		return ""
	}

	column, ok := fields[strings.ToLower(parts[0])]
	if !ok {
		return ""
	}

	direction := "DESC"
	if len(parts) > 1 {
		switch strings.ToUpper(parts[1]) {
		case "ASC":
			direction = "ASC"
		case "DESC":
			direction = "DESC"
		default:
			direction = "DESC"
		}
	}

	return column + " " + direction
}
