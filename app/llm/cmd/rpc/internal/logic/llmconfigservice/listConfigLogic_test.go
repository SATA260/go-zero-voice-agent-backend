package llmconfigservicelogic

import (
	"context"
	"testing"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/config"
	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestListConfigLogic_ListConfig(t *testing.T) {
	var c config.Config
	conf.MustLoad("../../etc/llm.yaml", &c)
	svcCtx := svc.NewServiceContext(c)
	ctx := context.Background()
	logic := NewListConfigLogic(ctx, svcCtx)

	req := &pb.ListConfigReq{
		PageQuery: &pb.PageQuery{
			Page:     1,
			PageSize: 10,
		},
	}

	_, err := logic.ListConfig(req)
	if err != nil {
		t.Errorf("ListConfig() error = %v", err)
		return
	}
}
