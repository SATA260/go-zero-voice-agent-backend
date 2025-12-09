package docservicelogic

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go-zero-voice-agent/app/rag/cmd/rpc/internal/ragclient"
	"go-zero-voice-agent/app/rag/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/rag/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ListChunksLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListChunksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListChunksLogic {
	return &ListChunksLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListChunksLogic) ListChunks(in *pb.ListChunksReq) (*pb.ListChunksResp, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request must not be nil")
	}

	userID := in.GetUserId()
	if userID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	page := int64(1)
	pageSize := int64(20)
	orderBy := ""
	sort := "desc"

	if pq := in.GetPageQuery(); pq != nil {
		if pq.Page > 0 {
			page = pq.Page
		}
		if pq.PageSize > 0 {
			pageSize = pq.PageSize
		}
		if order := strings.TrimSpace(pq.GetOrderBy()); order != "" {
			splits := strings.Split(order, " ")
			if len(splits) != 2 {
				return nil, status.Error(codes.InvalidArgument, "invalid order_by format")
			}
			orderBy = splits[0]
			sort = strings.ToLower(splits[1])
			if sort != "asc" && sort != "desc" {
				return nil, status.Error(codes.InvalidArgument, "invalid sort order")
			}
		}
	}

	if pageSize > 200 {
		pageSize = 200
	}

	params := ragclient.ListChunksParams{
		Page:     int(page),
		PageSize: int(pageSize),
		FileID:   in.GetFileId(),
		OrderBy:  orderBy,
		Sort:     sort,
	}

	resp, err := l.svcCtx.RagClient.ListChunks(l.ctx, strconv.FormatInt(userID, 10), &params)
	if err != nil {
		return nil, err
	}

	records := make([]*pb.DocumentRecord, 0, len(resp.Items))

	for _, item := range resp.Items {
		records = append(records, &pb.DocumentRecord{
			CustomId:    item.CustomID,
			PageContent: item.PageContent,
			Metadata:    convertMapString(item.Metadata),
		})
	}

	return &pb.ListChunksResp{
		Chunks: records,
		Total:  int64(resp.Total),
	}, nil
}

func convertMapString(input map[string]any) map[string]string {
	result := make(map[string]string)
	for k, v := range input {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}
