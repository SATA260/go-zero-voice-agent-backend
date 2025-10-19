package logic

import (
	"context"
	"net/smtp"
	"net/textproto"
	"time"

	"go-zero-voice-agent/app/usercenter/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/usercenter/cmd/rpc/pb"

	"github.com/jordan-wright/email"
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
	addr :=  host + ":" +  string(rune(port))
	username := l.svcCtx.Config.Email.Username
	password := l.svcCtx.Config.Email.Password

	err := e.Send(addr, smtp.PlainAuth("", username, password, host))

	if err != nil {
		return nil, err
	}

	return &pb.SendEmailResp{
		SendAt: time.Now().Unix(),
	}, nil
}
