// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"go-zero-voice-agent/app/usercenter/cmd/api/internal/svc"
	"go-zero-voice-agent/app/usercenter/cmd/api/internal/types"
	"go-zero-voice-agent/app/usercenter/cmd/rpc/usercenter"
	"go-zero-voice-agent/app/usercenter/model"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// register
func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterReq) (resp *types.RegisterResp, err error) {
	registerResp, err := l.svcCtx.UsercenterRpc.Register(l.ctx, &usercenter.RegisterReq{
		Email:      req.Email,
		Password:   req.Password,
		VerifyCode: req.Code,
		AuthKey:    req.Email,
		AuthType:   model.UserAuthTypeEmail,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Fail to register user, email:%s", req.Email)
	}

	return &types.RegisterResp{
		AccessToken:  registerResp.AccessToken,
		AccessExpire: registerResp.AccessExpire,
		RefreshAfter: registerResp.RefreshAfter,
	}, nil
}
