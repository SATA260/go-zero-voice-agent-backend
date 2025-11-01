package logic

import (
	"context"
	"testing"

	"go-zero-voice-agent/app/llmservice/cmd/rpc/internal/config"
	"go-zero-voice-agent/app/llmservice/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llmservice/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestDeleteConfigLogic_DeleteConfig(t *testing.T) {
	var c config.Config
	conf.MustLoad("../../etc/llmservice.yaml", &c)
	svcCtx := svc.NewServiceContext(c)
	ctx := context.Background()

	// First create a config to delete
	createLogic := NewCreateConfigLogic(ctx, svcCtx)
	createReq := &pb.CreateConfigReq{
		UserId: 1,
		Name:   "test-config-for-delete",
	}
	createResp, err := createLogic.CreateConfig(createReq)
	if err != nil {
		t.Fatalf("Failed to create config for delete test: %v", err)
	}

	logic := NewDeleteConfigLogic(ctx, svcCtx)

	req := &pb.DeleteConfigReq{
		Id: createResp.Id,
	}

	_, err = logic.DeleteConfig(req)
	if err != nil {
		t.Errorf("DeleteConfig() error = %v", err)
		return
	}
}
