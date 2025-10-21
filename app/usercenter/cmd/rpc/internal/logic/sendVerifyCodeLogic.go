package logic

import (
	"context"
	"fmt"
	"math/rand"

	"go-zero-voice-agent/app/usercenter/cmd/rpc/internal/consts"
	"go-zero-voice-agent/app/usercenter/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/usercenter/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendVerifyCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendVerifyCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendVerifyCodeLogic {
	return &SendVerifyCodeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendVerifyCodeLogic) SendVerifyCode(in *pb.SendVerifyCodeReq) (*pb.SendVerifyCodeResp, error) {
	code := rand.Intn(1000000)
	codeStr := fmt.Sprintf("%06d", code)


	l.svcCtx.RedisClient.Setex(consts.GetRegisterVerifyCodeCacheKey(in.Email), codeStr, int(in.AccessExpire))
	sendEmailLogic := NewSendEmailLogic(l.ctx, l.svcCtx)
	sendEmailReq := &pb.SendEmailReq{
		To:      in.Email,
		From:    l.svcCtx.Config.Email.Username,
		Subject: "【用户注册】验证码",
		Text:    fmt.Sprintf("您的验证码是：%s", codeStr),
	}
	logx.Infof("Store verify code %s for email %s in redis", codeStr, in.Email)


	sendEmailResp, err := sendEmailLogic.SendEmail(sendEmailReq)
	if err != nil {
		return nil, err
	}

	logx.Infof("send verify code %s to email %s", codeStr, in.Email)

	return &pb.SendVerifyCodeResp{
		SendAt: sendEmailResp.SendAt,
	}, nil
}
