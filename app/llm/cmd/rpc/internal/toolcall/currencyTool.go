package toolcall

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go-zero-voice-agent/app/llm/pkg/consts"
)

type CurrencyTool struct{}

func NewCurrencyTool() *CurrencyTool {
	return &CurrencyTool{}
}

func (t *CurrencyTool) Name() string {
	return "currency_converter"
}

func (t *CurrencyTool) Description() string {
	return "汇率转换工具。支持的货币：USD(美元), CNY(人民币), EUR(欧元), JPY(日元), GBP(英镑)。"
}

func (t *CurrencyTool) ArgumentsJson() string {
	return `{
  "type": "object",
  "properties": {
    "amount": { "type": "number", "description": "需要转换的金额" },
    "from": { "type": "string", "description": "源货币代码 (例如: USD)" },
    "to": { "type": "string", "description": "目标货币代码 (例如: CNY)" }
  },
  "required": ["amount", "from", "to"]
}`
}

func (t *CurrencyTool) Scope() string {
	return consts.TOOL_CALLING_SCOPE_SERVER
}

func (t *CurrencyTool) RequiresConfirmation() bool {
	return false
}

func (t *CurrencyTool) Execute(ctx context.Context, argsJson string) (string, error) {
	var params struct {
		Amount float64 `json:"amount"`
		From   string  `json:"from"`
		To     string  `json:"to"`
	}
	if err := json.Unmarshal([]byte(argsJson), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	from := strings.ToUpper(params.From)
	to := strings.ToUpper(params.To)

	// 简单 Mock 汇率 (基准 USD)
	// 实际生产中应调用实时汇率 API
	rates := map[string]float64{
		"USD": 1.0,
		"CNY": 7.25,
		"EUR": 0.92,
		"JPY": 150.0,
		"GBP": 0.79,
		"HKD": 7.82,
	}

	rateFrom, ok1 := rates[from]
	rateTo, ok2 := rates[to]

	if !ok1 || !ok2 {
		return "", fmt.Errorf("unsupported currency pair: %s -> %s. Supported: USD, CNY, EUR, JPY, GBP, HKD", from, to)
	}

	// 转换逻辑: Amount / RateFrom * RateTo
	result := params.Amount / rateFrom * rateTo

	return fmt.Sprintf("%.2f %s = %.2f %s (参考汇率: 1 %s = %.4f %s)",
		params.Amount, from, result, to, from, rateTo/rateFrom, to), nil
}
