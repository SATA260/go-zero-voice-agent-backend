package toolcall

import (
	"context"
	"encoding/json"
	"fmt"
	"go-zero-voice-agent/app/llm/pkg/consts"
	"io"
	"net/http"
	"net/url"
)

type WeatherTool struct{}

func NewWeatherTool() *WeatherTool {
	return &WeatherTool{}
}

func (t *WeatherTool) Name() string {
	return "weather_tool"
}

func (t *WeatherTool) Description() string {
	return "天气工具,用于获取指定省市区的天气情况"
}

func (t *WeatherTool) ArgumentsJson() string {
	return `{
  "type": "object",
  "properties": {
    "province": {
      "type": "string",
      "description": "省份，例如：四川"
    },
    "city": {
      "type": "string",
      "description": "城市，例如：成都"
    },
    "county": {
      "type": "string",
      "description": "区县，例如：成华区"
    }
  },
  "required": ["province", "city", "county"]
}`
}

func (t *WeatherTool) RequiresConfirmation() bool {
	return false
}

func (t *WeatherTool) Scope() string {
	return consts.TOOL_CALLING_SCOPE_SERVER
}

type WeatherResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Observe struct {
			Degree            string `json:"degree"`
			Humidity          string `json:"humidity"`
			Precipitation     string `json:"precipitation"`
			Pressure          string `json:"pressure"`
			UpdateTime        string `json:"update_time"`
			Weather           string `json:"weather"`
			WeatherCode       string `json:"weather_code"`
			WeatherShort      string `json:"weather_short"`
			WindDirection     string `json:"wind_direction"`
			WindPower         string `json:"wind_power"`
			WindDirectionName string `json:"wind_direction_name"`
		} `json:"observe"`
	} `json:"data"`
}

func (t *WeatherTool) Execute(ctx context.Context, argsJson string) (string, error) {
	var args struct {
		Province string `json:"province"`
		City     string `json:"city"`
		County   string `json:"county"`
	}
	if err := json.Unmarshal([]byte(argsJson), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if args.Province == "" || args.City == "" || args.County == "" {
		return "", fmt.Errorf("province, city, and county are required")
	}

	apiURL := "https://wis.qq.com/weather/common"
	params := url.Values{}
	params.Add("source", "pc")
	params.Add("weather_type", "observe")
	params.Add("province", args.Province)
	params.Add("city", args.City)
	params.Add("county", args.County)

	resp, err := http.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return "", fmt.Errorf("failed to fetch weather: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	var weatherResp WeatherResponse
	if err := json.Unmarshal(body, &weatherResp); err != nil {
		return "", fmt.Errorf("failed to parse weather response: %v", err)
	}

	if weatherResp.Status != 200 {
		return "", fmt.Errorf("weather api error: %s", weatherResp.Message)
	}

	obs := weatherResp.Data.Observe
	result := fmt.Sprintf("当前%s%s%s的天气情况：\n天气：%s\n温度：%s℃\n湿度：%s%%\n风向：%s\n风力：%s级\n气压：%shPa\n更新时间：%s",
		args.Province, args.City, args.County,
		obs.Weather, obs.Degree, obs.Humidity, obs.WindDirectionName, obs.WindPower, obs.Pressure, obs.UpdateTime)

	return result, nil
}
