package toolcall

import (
	"context"
	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"time"
)

type TimeTool struct {
	svcCtx *svc.ServiceContext
}

func NewTimeTool(svcCtx *svc.ServiceContext) *TimeTool {
	return &TimeTool{
		svcCtx: svcCtx,
	}
}

func (t *TimeTool) Name() string {
	return "time_tool"
}

func (t *TimeTool) Description() string {
	return "时间工具,用于获取当前时间"
}

func (t *TimeTool) ArgumentsJson() string {
	return `{}`
}

func (t *TimeTool) Execute(ctx context.Context, argsJson string) (string, error) {
	currentTime := "当前时间是: " + time.Now().Format("2006-01-02 15:04:05")
	return currentTime, nil
}