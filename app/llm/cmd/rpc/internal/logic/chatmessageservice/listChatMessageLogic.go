package chatmessageservicelogic

import (
	"context"
	"strings"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"

	"github.com/Masterminds/squirrel"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListChatMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListChatMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListChatMessageLogic {
	return &ListChatMessageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListChatMessageLogic) ListChatMessage(in *pb.ListChatMessageReq) (*pb.ListChatMessageResp, error) {
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

		orderBy = sanitizeMessageOrderBy(strings.TrimSpace(pq.GetOrderBy()))
	}

	builder := l.svcCtx.ChatMessageModel.SelectBuilder()

	if filter := in.GetFilter(); filter != nil {
		if filter.GetId() > 0 {
			builder = builder.Where(squirrel.Eq{"id": filter.GetId()})
		}
		if filter.GetSessionId() > 0 {
			builder = builder.Where(squirrel.Eq{"session_id": filter.GetSessionId()})
		}
		if role := strings.TrimSpace(filter.GetRole()); role != "" {
			builder = builder.Where(squirrel.Eq{"role": role})
		}
	}

	records, total, err := l.svcCtx.ChatMessageModel.FindPageListByPageWithTotal(l.ctx, builder, page, pageSize, orderBy)
	if err != nil {
		return nil, errors.Wrapf(err, "list chat messages failed, req: %+v", in)
	}

	messages := make([]*pb.ChatMessage, 0, len(records))
	for _, record := range records {
		messages = append(messages, chatMessageToPb(record))
	}

	return &pb.ListChatMessageResp{
		Total:    total,
		Messages: messages,
	}, nil
}

func sanitizeMessageOrderBy(orderBy string) string {
	orderBy = strings.TrimSpace(orderBy)
	if orderBy == "" {
		return ""
	}

	allowed := map[string]string{
		"id":          "id",
		"create_time": "create_time",
		"update_time": "update_time",
		"session_id":  "session_id",
	}

	parts := strings.Fields(orderBy)
	if len(parts) == 0 {
		return ""
	}

	column, ok := allowed[strings.ToLower(parts[0])]
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
		}
	}

	return column + " " + direction
}
