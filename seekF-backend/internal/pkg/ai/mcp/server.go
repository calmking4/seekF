package mcp

import (
	"context"
	"os"
	"sync"

	"seekF-backend/internal/pkg/ai/mcp/tool"
	"seekF-backend/internal/pkg/zlog"

	"github.com/mark3labs/mcp-go/server"
)

var (
	mcpServer   *server.MCPServer
	weatherTool *tool.WeatherTool
	initOnce    sync.Once
)

func InitMCPServer() error {
	var initErr error
	initOnce.Do(func() {
		weatherTool = tool.NewWeatherTool()

		mcpServer = server.NewMCPServer(
			"seekF-weather",
			"1.0.0",
			server.WithToolCapabilities(false),
			server.WithRecovery(),
		)

		mcpServer.AddTool(weatherTool.GetWeatherTool(), weatherTool.HandleWeatherRequest)

		zlog.Info("MCP server initialized with weather tool")
	})
	return initErr
}

func GetMCPServer() *server.MCPServer {
	if mcpServer == nil {
		if err := InitMCPServer(); err != nil {
			zlog.Error("failed to init MCP server: " + err.Error())
			return nil
		}
	}
	return mcpServer
}

func ServeStdio(ctx context.Context) error {
	srv := GetMCPServer()
	if srv == nil {
		return nil
	}

	zlog.Info("Starting MCP server with stdio transport")
	return server.ServeStdio(srv)
}

func ServeStdioWithOptions(ctx context.Context) error {
	srv := GetMCPServer()
	if srv == nil {
		return nil
	}

	stdioServer := server.NewStdioServer(srv)
	return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
}
