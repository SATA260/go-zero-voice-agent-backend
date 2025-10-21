package logic

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"net/textproto"
	"time"

	"go-zero-voice-agent/app/usercenter/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/usercenter/cmd/rpc/pb"

	"github.com/jordan-wright/email"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type SendEmailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendEmailLogic {
	return &SendEmailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendEmailLogic) SendEmail(in *pb.SendEmailReq) (*pb.SendEmailResp, error) {
	e := &email.Email{
		To:      []string{in.To},
		From:    in.From,
		Subject: in.Subject,
		Text:    []byte(in.Text),
		HTML:    []byte(in.Html),
		Headers: textproto.MIMEHeader{},
	}

	host := l.svcCtx.Config.Email.Host
	port := l.svcCtx.Config.Email.Port
	addr :=  fmt.Sprintf("%s:%d", host, port)
	username := l.svcCtx.Config.Email.Username
	password := l.svcCtx.Config.Email.Password

	logx.Infof("%s start to send email to %s", in.From, in.To)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}
	err := e.SendWithTLS(addr, smtp.PlainAuth("", username, password, host), tlsConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to send email to %s", in.To)
		}
		logx.Infof("Successfully sent email to %s", in.To)

	return &pb.SendEmailResp{
		SendAt: time.Now().Unix(),
	}, nil
}
