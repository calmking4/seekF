package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"

	"github.com/mark3labs/mcp-go/mcp"
)

type ExchangeRateTool struct {
	apiKey string
}

func NewExchangeRateTool() *ExchangeRateTool {
	return &ExchangeRateTool{
		apiKey: configs.GetConfig().ExchangeRateConfig.APIKey,
	}
}

// GetExchangeRateTool 获取汇率查询工具
func (t *ExchangeRateTool) GetExchangeRateTool() mcp.Tool {
	return mcp.NewTool(
		"get_exchange_rate",
		mcp.WithDescription("查询两种货币之间的实时汇率。适用于询问货币兑换比率、汇率换算等场景。"),
		mcp.WithString("base_currency",
			mcp.Required(),
			mcp.Description("基础货币的ISO 4217代码，如：USD（美元）、EUR（欧元）、CNY（人民币）、GBP（英镑）、JPY（日元）等。"),
		),
		mcp.WithString("target_currency",
			mcp.Required(),
			mcp.Description("目标货币的ISO 4217代码，如：USD（美元）、EUR（欧元）、CNY（人民币）、GBP（英镑）、JPY（日元）等。"),
		),
	)
}

// HandleExchangeRateRequest 处理汇率查询请求
func (t *ExchangeRateTool) HandleExchangeRateRequest(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultText("参数解析失败"), nil
	}

	baseCurrency, ok := arguments["base_currency"].(string)
	if !ok || baseCurrency == "" {
		return mcp.NewToolResultText("请提供基础货币代码（如：USD、EUR、CNY）"), nil
	}

	targetCurrency, ok := arguments["target_currency"].(string)
	if !ok || targetCurrency == "" {
		return mcp.NewToolResultText("请提供目标货币代码（如：USD、EUR、CNY）"), nil
	}

	// 标准化货币代码为大写
	baseCurrency = strings.ToUpper(strings.TrimSpace(baseCurrency))
	targetCurrency = strings.ToUpper(strings.TrimSpace(targetCurrency))

	rateInfo, err := t.queryExchangeRate(ctx, baseCurrency, targetCurrency)
	if err != nil {
		zlog.Error("query exchange rate failed: " + err.Error())
		return mcp.NewToolResultText("查询汇率失败: " + err.Error()), nil
	}

	return mcp.NewToolResultText(rateInfo), nil
}

// queryExchangeRate 查询两种货币之间的汇率
func (t *ExchangeRateTool) queryExchangeRate(ctx context.Context, baseCurrency, targetCurrency string) (string, error) {
	if t.apiKey == "" {
		return "", fmt.Errorf("exchange rate api key not configured")
	}

	// 构建ExchangeRate-API请求URL
	apiURL := fmt.Sprintf("https://v6.exchangerate-api.com/v6/%s/pair/%s/%s",
		t.apiKey, baseCurrency, targetCurrency)

	// 创建HTTP请求并设置上下文
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}

	// 初始化HTTP客户端并设置超时时间
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	// 定义响应数据结构体
	var result struct {
		Result             string  `json:"result"`
		Documentation      string  `json:"documentation"`
		TermsOfUse         string  `json:"terms_of_use"`
		TimeLastUpdateUnix int64   `json:"time_last_update_unix"`
		TimeLastUpdateUTC  string  `json:"time_last_update_utc"`
		TimeNextUpdateUnix int64   `json:"time_next_update_unix"`
		TimeNextUpdateUTC  string  `json:"time_next_update_utc"`
		BaseCode           string  `json:"base_code"`
		TargetCode         string  `json:"target_code"`
		ConversionRate     float64 `json:"conversion_rate"`
		ErrorType          string  `json:"error-type"`
	}

	// 解析API响应JSON数据
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// 检查API返回的错误
	if result.Result == "error" {
		errorMsg := getExchangeRateErrorMessage(result.ErrorType)
		return "", fmt.Errorf("API error: %s", errorMsg)
	}

	// 获取货币中文名称
	baseName := getCurrencyName(baseCurrency)
	targetName := getCurrencyName(targetCurrency)

	// 格式化汇率信息输出
	rateInfo := fmt.Sprintf(`💱 汇率查询结果
基础货币: %s (%s)
目标货币: %s (%s)
汇率: 1 %s = %.4f %s
更新时间: %s

数据来源: ExchangeRate-API`,
		baseCurrency, baseName,
		targetCurrency, targetName,
		baseCurrency, result.ConversionRate, targetCurrency,
		result.TimeLastUpdateUTC)

	return rateInfo, nil
}

// getCurrencyName 获取货币的中文名称
func getCurrencyName(code string) string {
	currencyNames := map[string]string{
		"USD": "美元",
		"EUR": "欧元",
		"CNY": "人民币",
		"GBP": "英镑",
		"JPY": "日元",
		"KRW": "韩元",
		"HKD": "港币",
		"TWD": "新台币",
		"SGD": "新加坡元",
		"AUD": "澳元",
		"CAD": "加元",
		"CHF": "瑞士法郎",
		"MYR": "马来西亚林吉特",
		"THB": "泰铢",
		"RUB": "俄罗斯卢布",
		"INR": "印度卢比",
		"BRL": "巴西雷亚尔",
		"ZAR": "南非兰特",
		"AED": "阿联酋迪拉姆",
		"SAR": "沙特里亚尔",
		"NZD": "新西兰元",
		"SEK": "瑞典克朗",
		"NOK": "挪威克朗",
		"DKK": "丹麦克朗",
		"PLN": "波兰兹罗提",
		"PHP": "菲律宾比索",
		"IDR": "印尼盾",
		"VND": "越南盾",
		"EGP": "埃及镑",
		"TRY": "土耳其里拉",
		"MXN": "墨西哥比索",
		"CLP": "智利比索",
		"COP": "哥伦比亚比索",
		"PEN": "秘鲁索尔",
		"ARS": "阿根廷比索",
	}

	if name, ok := currencyNames[code]; ok {
		return name
	}
	return code
}

// getExchangeRateErrorMessage 获取汇率API错误信息的中文翻译
func getExchangeRateErrorMessage(errorType string) string {
	errorMessages := map[string]string{
		"unsupported-code":  "不支持的货币代码",
		"malformed-request": "请求格式错误",
		"invalid-key":       "API密钥无效",
		"inactive-account":  "账户未激活",
		"quota-reached":     "API请求配额已用尽",
	}

	if msg, ok := errorMessages[errorType]; ok {
		return msg
	}
	return "未知错误: " + errorType
}
