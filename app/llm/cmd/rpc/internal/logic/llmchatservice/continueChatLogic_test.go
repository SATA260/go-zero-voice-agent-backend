package llmchatservicelogic

import (
	"context"
	"os"
	"testing"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/config"
	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestContinueChatLogic_ContinueChat(t *testing.T) {
	apiKey := os.Getenv("API_KEY")
	baseUrl := os.Getenv("BASE_URL")
	if apiKey == "" || baseUrl == "" {
		t.Skip("API_KEY or BASE_URL environment variables not set")
	}

	var c config.Config
	conf.MustLoad("../../etc/llm.yaml", &c)
	svcCtx := svc.NewServiceContext(c)
	ctx := context.Background()
	logic := NewContinueChatLogic(ctx, svcCtx)

	req := &pb.ContinueChatReq{
		Id: "test-session",
		LlmConfig: &pb.LlmConfig{
			Model:         "qwen-flash",
			ApiKey:        apiKey,
			BaseUrl:       baseUrl,
			ContentLength: 10,
		},
	}

	_, err := logic.ContinueChat(req)
	if err != nil {
		t.Errorf("ContinueChat() error = %v", err)
		return
	}
}
