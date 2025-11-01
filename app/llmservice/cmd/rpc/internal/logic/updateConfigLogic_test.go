package logic

import (
	"context"
	"testing"

	"go-zero-voice-agent/app/llmservice/cmd/rpc/internal/config"
	"go-zero-voice-agent/app/llmservice/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llmservice/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestUpdateConfigLogic_UpdateConfig(t *testing.T) {
	var c config.Config
	conf.MustLoad("../../etc/llmservice.yaml", &c)
	svcCtx := svc.NewServiceContext(c)
	ctx := context.Background()

	// First create a config to update
	createLogic := NewCreateConfigLogic(ctx, svcCtx)
	createReq := &pb.CreateConfigReq{
		UserId: 1,
		Name:   "test-config-for-update",
	}
	createResp, err := createLogic.CreateConfig(createReq)
	if err != nil {
		t.Fatalf("Failed to create config for update test: %v", err)
	}

	logic := NewUpdateConfigLogic(ctx, svcCtx)

	req := &pb.UpdateConfigReq{
		Id:                createResp.Id,
		Name:              "updated-test-config",
		Description:       "updated description",
		UserId:            1,
		BaseUrl:           "https://updated.example.com",
		ApiKey:            "updated-api-key",
		Model:             "updated-model",
		Stream:            1,
		Temperature:       0.5,
		TopP:              0.9,
		TopK:              50,
		EnableThinking:    1,
		RepetitionPenalty: 1.0,
		PresencePenalty:   0.1,
		MaxTokens:         1000,
		Seed:              42,
		EnableSearch:      1,
		ContextLength:     2000,
	}

	_, err = logic.UpdateConfig(req)
	if err != nil {
		t.Errorf("UpdateConfig() error = %v", err)
		return
	}
}
