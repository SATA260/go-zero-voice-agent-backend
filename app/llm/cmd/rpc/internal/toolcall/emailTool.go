package toolcall

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/smtp"
	"time"

	"github.com/jordan-wright/email"
)

type emailTool struct {
	// 实际场景中，SMTP 配置应该从 config 或 svcCtx 中获取
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
}

type EmailToolParams struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}

// NewEmailTool 初始化邮件工具
// 建议后续将配置参数改为从 svcCtx.Config 中读取
func NewEmailTool(host, port, username, password string) *emailTool {
	return &emailTool{
		// 这里使用硬编码仅作示例，请替换为你的真实 SMTP 配置或从配置中心读取
		smtpHost:     host,
		smtpPort:     port,
		smtpUsername: username,
		smtpPassword: password,
	}
}

func (t *emailTool) Name() string {
	return "send_email"
}

func (t *emailTool) Description() string {
	return "发送电子邮件。参数包括收件人列表(to)、主题(subject)和正文(body)。"
}

func (t *emailTool) ArgumentsJson() string {
	return `{
		"to": "收件人列表，字符串数组类型",
		"subject": "邮件主题，字符串类型",
		"body": "邮件正文，字符串类型"
	}`
}

func (t *emailTool) Execute(ctx context.Context, argsJson string) (string, error) {
	var params EmailToolParams
	if err := json.Unmarshal([]byte(argsJson), &params); err != nil {
		return "", fmt.Errorf("解析邮件参数失败: %w", err)
	}

	if len(params.To) == 0 {
		return "", fmt.Errorf("收件人不能为空")
	}

	e := email.NewEmail()
	e.From = fmt.Sprintf("Voice Agent <%s>", t.smtpUsername)
	e.To = params.To
	e.Subject = params.Subject
	e.Text = []byte(params.Body)

	addr := fmt.Sprintf("%s:%s", t.smtpHost, t.smtpPort)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         t.smtpHost,
	}
	if err := e.SendWithTLS(addr, smtp.PlainAuth("", t.smtpUsername, t.smtpPassword, t.smtpHost), tlsConfig); err != nil {
		return "", fmt.Errorf("发送邮件失败: %w", err)
	}

	return fmt.Sprintf("邮件已成功发送给 %v, 发送时间 %s", params.To, time.Now().Format("2006-01-02 15:04:05")), nil
}
