package ragservicelogic

import (
	"context"
	"fmt"
	"strings"

	"go-zero-voice-agent/app/rag/cmd/rpc/internal/ragclient"
	"go-zero-voice-agent/app/rag/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/rag/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type QueryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryLogic {
	return &QueryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *QueryLogic) Query(in *pb.QueryReq) (*pb.QueryResp, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request must not be nil")
	}

	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	queryText := strings.TrimSpace(in.GetQuery())
	if queryText == "" {
		return nil, status.Error(codes.InvalidArgument, "query text is required")
	}

	req := &ragclient.QueryRequest{
		Query:  queryText,
		FileID: strings.TrimSpace(in.GetFileId()),
		TopK:   int(in.GetTopK()),
	}

	if req.TopK < 0 {
		req.TopK = 0
	}

	if entityID := strings.TrimSpace(in.GetEntityId()); entityID != "" {
		req.EntityID = &entityID
	}

	resp, err := l.svcCtx.RagClient.Query(l.ctx, userID, req)
	if err != nil {
		l.Logger.Errorf("query rag service failed: %v", err)
		return nil, status.Error(codes.Internal, "rag service query failed")
	}

	return buildQueryResp(resp), nil
}

func buildQueryResp(resp *ragclient.QueryResponse) *pb.QueryResp {
	if resp == nil || len(resp.Results) == 0 {
		return &pb.QueryResp{}
	}

	out := make([]*pb.RetrievalResult, 0, len(resp.Results))
	for _, item := range resp.Results {
		entry := &pb.RetrievalResult{
			PageContent: item.PageContent,
			Score:       item.Score,
			Metadata:    map[string]string{},
		}

		if len(item.Metadata) > 0 {
			entry.Metadata = make(map[string]string, len(item.Metadata))
			for k, v := range item.Metadata {
				if k == "" {
					continue
				}
				entry.Metadata[k] = fmt.Sprint(v)
			}
		}

		out = append(out, entry)
	}

	return &pb.QueryResp{Results: out}
}
