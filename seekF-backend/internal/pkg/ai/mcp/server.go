package mcp

import (
	"sync"

	"seekF-backend/internal/pkg/ai/mcp/tool"
	"seekF-backend/internal/pkg/zlog"

	"github.com/mark3labs/mcp-go/server"
)

var (
	mcpServer        *server.MCPServer
	weatherTool      *tool.WeatherTool
	exchangeRateTool *tool.ExchangeRateTool
	initOnce         sync.Once
)

// InitMCPServer 初始化MCP服务器实例并注册天气工具
func InitMCPServer() error {
	var initErr error
	initOnce.Do(func() {
		weatherTool = tool.NewWeatherTool()
		exchangeRateTool = tool.NewExchangeRateTool()

		mcpServer = server.NewMCPServer(
			"seekF-weather",
			"1.0.0",
			server.WithToolCapabilities(false),
			server.WithRecovery(),
		)

		mcpServer.AddTool(weatherTool.GetWeatherTool(), weatherTool.HandleWeatherRequest)
		mcpServer.AddTool(exchangeRateTool.GetExchangeRateTool(), exchangeRateTool.HandleExchangeRateRequest)

		zlog.Info("MCP server initialized with weather and exchange rate tools")
	})
	return initErr
}

// GetMCPServer 获取MCP服务器实例，如果尚未初始化则先进行初始化
func GetMCPServer() *server.MCPServer {
	if mcpServer == nil {
		if err := InitMCPServer(); err != nil {
			zlog.Error("failed to init MCP server: " + err.Error())
			return nil
		}
	}
	return mcpServer
}
