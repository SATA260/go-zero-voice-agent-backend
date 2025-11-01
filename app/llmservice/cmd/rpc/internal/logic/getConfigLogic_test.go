package logic

import (
	"context"
	"testing"

	"go-zero-voice-agent/app/llmservice/cmd/rpc/internal/config"
	"go-zero-voice-agent/app/llmservice/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llmservice/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestGetConfigLogic_GetConfig(t *testing.T) {
	var c config.Config
	conf.MustLoad("../../etc/llmservice.yaml", &c)
	svcCtx := svc.NewServiceContext(c)
	ctx := context.Background()

	// First create a config to get
	createLogic := NewCreateConfigLogic(ctx, svcCtx)
	createReq := &pb.CreateConfigReq{
		UserId: 1,
		Name:   "test-config-for-get",
	}
	createResp, err := createLogic.CreateConfig(createReq)
	if err != nil {
		t.Fatalf("Failed to create config for get test: %v", err)
	}

	logic := NewGetConfigLogic(ctx, svcCtx)

	req := &pb.GetConfigReq{
		Id: createResp.Id,
	}

	_, err = logic.GetConfig(req)
	if err != nil {
		t.Errorf("GetConfig() error = %v", err)
		return
	}
}
