package main

import (
	"context"
	"fmt"
	"os"

	"seekF-backend/internal/pkg/ai/mcp/tool"
	"seekF-backend/internal/pkg/zlog"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// 初始化日志
	zlog.Info("MCP Server (stdio) 正在启动...")

	// 创建工具实例
	weatherTool := tool.NewWeatherTool()
	exchangeRateTool := tool.NewExchangeRateTool()
	webSearchTool := tool.NewWebSearchTool()
	discoverPostsTool := tool.NewDiscoverPostsTool()

	// 创建 MCP Server
	mcpServer := server.NewMCPServer(
		"seekF-mcp-server",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	// 注册工具
	mcpServer.AddTool(weatherTool.GetWeatherTool(), weatherTool.HandleWeatherRequest)
	mcpServer.AddTool(exchangeRateTool.GetExchangeRateTool(), exchangeRateTool.HandleExchangeRateRequest)
	mcpServer.AddTool(webSearchTool.GetWebSearchTool(), webSearchTool.HandleWebSearchRequest)
	mcpServer.AddTool(discoverPostsTool.GetDiscoverPostsTool(), discoverPostsTool.HandleDiscoverPostsRequest)

	zlog.Info("MCP Server 工具注册完成，包含天气、汇率、网页搜索和帖子搜索工具")

	// 使用 stdio 传输启动服务器
	// 从 stdin 读取 JSON-RPC 消息，将响应写入 stdout
	stdioServer := server.NewStdioServer(mcpServer)

	zlog.Info("MCP Server (stdio) 开始监听 stdin/stdout...")

	// 启动服务器（阻塞运行）
	ctx := context.Background()
	if err := stdioServer.Listen(ctx, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "MCP Server 启动失败: %v\n", err)
		os.Exit(1)
	}
}
