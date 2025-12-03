package docservicelogic

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

type FetchDocumentsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFetchDocumentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FetchDocumentsLogic {
	return &FetchDocumentsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *FetchDocumentsLogic) FetchDocuments(in *pb.FetchDocumentsReq) (*pb.FetchDocumentsResp, error) {
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

	resp, err := l.svcCtx.RagClient.FetchDocuments(l.ctx, userID, cleanedIDs)
	if err != nil {
		l.Logger.Errorf("fetch rag documents failed: %v", err)
		return nil, status.Error(codes.Internal, "rag service fetch documents failed")
	}

	return buildFetchDocumentsResp(resp), nil
}

func buildFetchDocumentsResp(resp *ragclient.DocumentsResponse) *pb.FetchDocumentsResp {
	if resp == nil || len(resp.Documents) == 0 {
		return &pb.FetchDocumentsResp{}
	}

	docs := make([]*pb.DocumentRecord, 0, len(resp.Documents))
	for _, item := range resp.Documents {
		record := &pb.DocumentRecord{
			PageContent: item.PageContent,
		}

		if len(item.Metadata) > 0 {
			record.Metadata = make(map[string]string, len(item.Metadata))
			for k, v := range item.Metadata {
				if strings.TrimSpace(k) == "" {
					continue
				}
				record.Metadata[k] = fmt.Sprint(v)
			}
		}

		docs = append(docs, record)
	}

	return &pb.FetchDocumentsResp{Documents: docs}
}
