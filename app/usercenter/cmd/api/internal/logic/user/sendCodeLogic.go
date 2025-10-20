// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"go-zero-voice-agent/app/usercenter/cmd/api/internal/svc"
	"go-zero-voice-agent/app/usercenter/cmd/api/internal/types"
	"go-zero-voice-agent/app/usercenter/cmd/rpc/usercenter"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type SendCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// send code
func NewSendCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendCodeLogic {
	return &SendCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendCodeLogic) SendCode(req *types.SendCodeReq) (resp *types.SendCodeResp, err error) {
	// 发送验证码
	if _, err := l.svcCtx.UsercenterRpc.SendVerifyCode(l.ctx, &usercenter.SendVerifyCodeReq{
		Email:        req.Email,
		AccessExpire: 30 * 60, // 30分钟
	}); err != nil {
		return nil, errors.Wrapf(err, "Fail to send verifyCode, email:%s", req.Email)
	}

	return &types.SendCodeResp{
		IsSuccess: true,
	}, nil
}
