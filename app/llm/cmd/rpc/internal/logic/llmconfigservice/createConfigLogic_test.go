package llmconfigservicelogic

import (
	"context"
	"testing"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/config"
	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestCreateConfigLogic_CreateConfig(t *testing.T) {
	var c config.Config
	conf.MustLoad("../../etc/llm.yaml", &c)
	svcCtx := svc.NewServiceContext(c)
	ctx := context.Background()
	logic := NewCreateConfigLogic(ctx, svcCtx)

	req := &pb.CreateConfigReq{
		UserId: 1,
		Name:   "test-config",
	}

	_, err := logic.CreateConfig(req)
	if err != nil {
		t.Errorf("CreateConfig() error = %v", err)
		return
	}
}
