package mcp

import (
	"context"
	"fmt"
	"sync"

	"seekF-backend/internal/pkg/zlog"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

var (
	einoTools []tool.BaseTool
	toolsOnce sync.Once
	toolsErr  error
)

func GetInProcessTools(ctx context.Context) ([]tool.BaseTool, error) {
	toolsOnce.Do(func() {
		mcpServer := GetMCPServer()
		if mcpServer == nil {
			toolsErr = GetLastInitError()
			return
		}

		mcpClient, err := client.NewInProcessClient(mcpServer)
		if err != nil {
			toolsErr = err
			zlog.Error("create in-process MCP client failed: " + err.Error())
			return
		}

		initRequest := mcp.InitializeRequest{}
		initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		initRequest.Params.ClientInfo = mcp.Implementation{
			Name:    "seekF-client",
			Version: "1.0.0",
		}

		_, err = mcpClient.Initialize(ctx, initRequest)
		if err != nil {
			toolsErr = err
			zlog.Error("initialize MCP client failed: " + err.Error())
			return
		}

		einoTools, err = mcpp.GetTools(ctx, &mcpp.Config{
			Cli: mcpClient,
		})
		if err != nil {
			toolsErr = err
			zlog.Error("get MCP tools failed: " + err.Error())
			return
		}

		zlog.Info(fmt.Sprintf("MCP tools initialized: %d tools", len(einoTools)))
	})
	return einoTools, toolsErr
}

func GetLastInitError() error {
	return nil
}

func GetMCPTools(ctx context.Context) ([]tool.BaseTool, error) {
	if len(einoTools) == 0 {
		return GetInProcessTools(ctx)
	}
	return einoTools, nil
}

func GetWeatherTool(ctx context.Context) (tool.BaseTool, error) {
	tools, err := GetMCPTools(ctx)
	if err != nil {
		return nil, err
	}

	for _, t := range tools {
		info, err := t.Info(ctx)
		if err != nil {
			continue
		}
		if info.Name == "get_weather" {
			return t, nil
		}
	}

	return nil, nil
}
