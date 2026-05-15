package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"

	"github.com/mark3labs/mcp-go/mcp"
)

// 复用连接，降低心知天气请求 TLS 握手开销
var seniverseHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

type WeatherTool struct {
	apiKey string
}

func NewWeatherTool() *WeatherTool {
	return &WeatherTool{
		apiKey: configs.GetConfig().SeniverseConfig.APIKey,
	}
}

// GetWeatherTool 获取天气查询工具
func (t *WeatherTool) GetWeatherTool() mcp.Tool {
	return mcp.NewTool(
		"get_weather",
		mcp.WithDescription("查询指定城市或地区的天气信息，包括天气现象、气温、风向风速、湿度等。适用于询问某个地方的天气情况。"),
		mcp.WithString("location",
			mcp.Required(),
			mcp.Description("要查询天气的城市或地区名称，如：北京、上海、杭州、成都、深圳、广州等。支持中文城市名、拼音或英文名称。"),
		),
	)
}

// HandleWeatherRequest 处理天气查询请求
func (t *WeatherTool) HandleWeatherRequest(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultText("参数解析失败"), nil
	}

	location, ok := arguments["location"].(string)
	if !ok || location == "" {
		return mcp.NewToolResultText("请提供要查询天气的城市名称"), nil
	}

	weather, err := t.queryWeather(ctx, location)
	if err != nil {
		zlog.Error("查询天气失败: " + err.Error())
		return mcp.NewToolResultText("查询天气失败: " + err.Error()), nil
	}

	return mcp.NewToolResultText(weather), nil
}

// queryWeather 查询指定位置的天气信息
func (t *WeatherTool) queryWeather(ctx context.Context, location string) (string, error) {
	if t.apiKey == "" {
		return "", fmt.Errorf("心知天气API密钥未配置")
	}

	// 构建心知天气API请求URL
	apiURL := fmt.Sprintf("https://api.seniverse.com/v3/weather/now.json?key=%s&location=%s&language=zh-Hans&unit=c",
		t.apiKey, url.QueryEscape(location)) //url.QueryEscape(location) 对 URL 查询参数进行编码 "北京" → "%E5%8C%97%E4%BA%AC"

	// 创建HTTP请求并设置上下文
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := seniverseHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() //在Go语言中，当发起HTTP请求后，需要手动关闭响应体以释放系统资源

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API请求失败，状态码: %d", resp.StatusCode)
	}

	// 定义响应数据结构体
	var result struct {
		Results []struct {
			Location struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Lat     string `json:"lat"`
				Lon     string `json:"lon"`
				Country string `json:"country"`
				Path    string `json:"path"`
			} `json:"location"`
			Now struct {
				Text        string `json:"text"`
				Code        string `json:"code"`
				Temperature string `json:"temperature"`
				WindScale   string `json:"wind_scale"`
				WindDir     string `json:"wind_direction"`
				Humidity    string `json:"humidity"`
				FeelsLike   string `json:"feels_like"`
				Pressure    string `json:"pressure"`
			} `json:"now"`
			LastUpdate string `json:"last_update"`
		} `json:"results"`
	}

	// 解析API响应JSON数据
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Results) == 0 {
		return "未找到该地区的天气信息，请尝试使用其他城市名称", nil
	}

	r := result.Results[0]
	now := r.Now

	// 格式化天气信息输出
	weatherInfo := fmt.Sprintf(`%s %s 天气/%s
🌡️ 气温: %s°C
💧 湿度: %s
🌬️ 风向: %s %s级
🏓 气压: %s hPa
🤒 体感: %s°C
🕐 更新时间: %s

数据来源: 心知天气`,
		r.Location.Country, r.Location.Name,
		now.Text, now.Temperature, now.Humidity,
		now.WindDir, now.WindScale, now.Pressure,
		now.FeelsLike, r.LastUpdate)

	return weatherInfo, nil
}
