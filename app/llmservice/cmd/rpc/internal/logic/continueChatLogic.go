package logic

import (
	"context"

	"go-zero-voice-agent/app/llmservice/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llmservice/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ContinueChatLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewContinueChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContinueChatLogic {
	return &ContinueChatLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ContinueChatLogic) ContinueChat(in *pb.ContinueChatReq) (*pb.ContinueChatResp, error) {
	// todo: add your logic here and delete this line

	return &pb.ContinueChatResp{}, nil
}
