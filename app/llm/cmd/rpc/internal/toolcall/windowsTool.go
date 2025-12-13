package toolcall

import (
	"context"
	"go-zero-voice-agent/app/llm/pkg/consts"
)

type WindowsTool struct{}

func NewWindowsTool() *WindowsTool {
	return &WindowsTool{}
}

func (t *WindowsTool) Name() string {
	return "windows_control"
}

func (t *WindowsTool) Description() string {
	return "Windows 系统控制工具。用于在用户的 Windows 电脑上执行操作，如打开应用、调节音量、系统控制等。该工具将在客户端执行。"
}

func (t *WindowsTool) ArgumentsJson() string {
	return `{
  "type": "object",
  "properties": {
    "method": {
      "type": "string",
      "description": "客户端预定义的方法名",
      "enum": ["OpenApp", "CloseApp", "SetVolume", "LockScreen", "Shutdown", "Restart", "MinimizeAll", "Screenshot", "SystemInfo"]
    },
    "params": {
      "type": "string",
      "description": "方法参数的 JSON 字符串。\n- OpenApp/CloseApp: '{\"name\": \"chrome\"}'\n- SetVolume: '{\"level\": 50}'\n- 其他无参方法: '{}'"
    }
  },
  "required": ["method", "params"]
}`
}

func (t *WindowsTool) Scope() string {
	return consts.TOOL_CALLING_SCOPE_CLIENT
}

func (t *WindowsTool) RequiresConfirmation() bool {
	return true
}

func (t *WindowsTool) Execute(ctx context.Context, argsJson string) (string, error) {
	return "Instruction sent to client for execution.", nil
}
