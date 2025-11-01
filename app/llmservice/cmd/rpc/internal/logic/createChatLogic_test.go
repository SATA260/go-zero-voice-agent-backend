package logic

import (
	"context"
	"os"
	"testing"

	"go-zero-voice-agent/app/llmservice/cmd/rpc/internal/config"
	"go-zero-voice-agent/app/llmservice/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llmservice/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestCreateChatLogic_CreateChat(t *testing.T) {
	apiKey := os.Getenv("API_KEY")
	baseUrl := os.Getenv("BASE_URL")
	if apiKey == "" || baseUrl == "" {
		t.Skip("API_KEY or BASE_URL environment variables not set")
	}

	var c config.Config
	conf.MustLoad("../../etc/llmservice.yaml", &c)
	svcCtx := svc.NewServiceContext(c)
	ctx := context.Background()
	logic := NewCreateChatLogic(ctx, svcCtx)

	req := &pb.CreateChatReq{
		LlmConfig: &pb.LlmConfig{
			Model:         "qwen-flash",
			ApiKey:        apiKey,
			BaseUrl:       baseUrl,
			ContentLength: 10,
		},
		Messages: []*pb.ChatMsg{
			{
				Role: "system",
				Content: "你叫lxl",
			},
			{
				Role:    "user",
				Content: "你好，请问你是谁",
			},
		},
	}

	_, err := logic.CreateChat(req)
	if err != nil {
		t.Errorf("CreateChat() error = %v", err)
		return
	}
}
