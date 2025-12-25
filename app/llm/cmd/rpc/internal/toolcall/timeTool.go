package toolcall

import (
	"context"
	"go-zero-voice-agent/app/llm/pkg/consts"
	"time"
)

type TimeTool struct{}

func NewTimeTool() *TimeTool {
	return &TimeTool{}
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

func (t *TimeTool) RequiresConfirmation() bool {
	return true
}

func (t *TimeTool) Scope() string {
	return consts.TOOL_CALLING_SCOPE_SERVER
}

func (t *TimeTool) Execute(ctx context.Context, argsJson string) (string, error) {
	currentTime := "当前时间是: " + time.Now().Format("2006-01-02 15:04:05")
	return currentTime, nil
}
