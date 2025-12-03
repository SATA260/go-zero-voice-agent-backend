package ragservicelogic

import (
	"context"
	"strconv"
	"strings"

	"go-zero-voice-agent/app/rag/cmd/rpc/internal/ragclient"
	"go-zero-voice-agent/app/rag/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/rag/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type QueryMultipleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewQueryMultipleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryMultipleLogic {
	return &QueryMultipleLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *QueryMultipleLogic) QueryMultiple(in *pb.QueryMultipleReq) (*pb.QueryResp, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request must not be nil")
	}

	userID := in.GetUserId()
	if userID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	queryText := strings.TrimSpace(in.GetQuery())
	if queryText == "" {
		return nil, status.Error(codes.InvalidArgument, "query text is required")
	}

	cleanedIDs := make([]string, 0, len(in.GetFileIds()))
	for _, id := range in.GetFileIds() {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			cleanedIDs = append(cleanedIDs, trimmed)
		}
	}

	if len(cleanedIDs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "file_ids must not be empty")
	}

	req := &ragclient.QueryMultipleRequest{
		Query:   queryText,
		FileIDs: cleanedIDs,
		TopK:    int(in.GetTopK()),
	}

	if req.TopK < 0 {
		req.TopK = 0
	}

	resp, err := l.svcCtx.RagClient.QueryMultiple(l.ctx, strconv.FormatInt(userID, 10), req)
	if err != nil {
		l.Logger.Errorf("query-multiple rag service failed: %v", err)
		return nil, status.Error(codes.Internal, "rag service query-multiple failed")
	}

	return buildQueryResp(resp), nil
}
