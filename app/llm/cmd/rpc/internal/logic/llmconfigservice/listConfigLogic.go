package llmconfigservicelogic

import (
	"context"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/pkg/errors"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListConfigLogic {
	return &ListConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListConfigLogic) ListConfig(in *pb.ListConfigReq) (*pb.ListConfigResp, error) {
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
		orderBy = sanitizeOrderBy(pq.GetOrderBy())
	}

	builder := l.svcCtx.ChatConfigModel.SelectBuilder()

	if filter := in.GetFilter(); filter != nil {
		if filter.GetId() > 0 {
			builder = builder.Where(squirrel.Eq{"id": filter.GetId()})
		}
		if filter.GetUserId() > 0 {
			builder = builder.Where(squirrel.Eq{"user_id": filter.GetUserId()})
		}
		if name := strings.TrimSpace(filter.GetName()); name != "" {
			builder = builder.Where("name LIKE ?", "%"+name+"%")
		}
		if desc := strings.TrimSpace(filter.GetDescription()); desc != "" {
			builder = builder.Where("description LIKE ?", "%"+desc+"%")
		}
	}

	records, total, err := l.svcCtx.ChatConfigModel.FindPageListByPageWithTotal(l.ctx, builder, page, pageSize, orderBy)
	if err != nil {
		return nil, errors.Wrapf(err, "list chat config failed, req: %+v", in)
	}

	configs := make([]*pb.ChatConfig, 0, len(records))
	for _, cfg := range records {
		configs = append(configs, chatConfigToPb(cfg))
	}

	return &pb.ListConfigResp{
		Total:   total,
		Configs: configs,
	}, nil
}

func sanitizeOrderBy(orderBy string) string {
	orderBy = strings.TrimSpace(orderBy)
	if orderBy == "" {
		return ""
	}

	fields := map[string]string{
		"id":          "id",
		"create_time": "create_time",
		"update_time": "update_time",
		"name":        "name",
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
