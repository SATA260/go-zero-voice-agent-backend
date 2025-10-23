// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"go-zero-voice-agent/app/usercenter/cmd/api/internal/svc"
	"go-zero-voice-agent/app/usercenter/cmd/api/internal/types"
	"go-zero-voice-agent/app/usercenter/cmd/rpc/usercenter"

	"github.com/zeromicro/go-zero/core/logx"
)

type VerifyTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// auth by token
func NewVerifyTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerifyTokenLogic {
	return &VerifyTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VerifyTokenLogic) VerifyToken(req *types.VerifyTokenReq) (resp *types.VerifyTokenResp, err error) {
	logx.Infof("VerifyToken req: %+v", req)

	verifyTokenResp, err := l.svcCtx.UsercenterRpc.VerifyToken(context.Background(), &usercenter.VerifyTokenReq{
		Token: req.Authorization,
	})
	if err != nil {
		return nil, err
	}

	return &types.VerifyTokenResp{
		UserId: verifyTokenResp.UserId,
	}, nil
}
