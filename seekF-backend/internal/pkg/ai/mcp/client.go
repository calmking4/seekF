package mcp

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"seekF-backend/internal/pkg/zlog"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

var (
	einoTools []tool.BaseTool // 存储已初始化的Eino工具列表
	toolsOnce sync.Once       // 确保工具只初始化一次的同步原语
	toolsErr  error           // 记录初始化过程中的错误
)

// GetInProcessTools 获取当前进程内的MCP工具实例
func GetInProcessTools(ctx context.Context) ([]tool.BaseTool, error) {
	toolsOnce.Do(func() {
		mcpServer := GetMCPServer()
		if mcpServer == nil {
			toolsErr = errors.New("MCP server is not initialized")
			return
		}

		// 创建与MCP服务器的进程内客户端连接
		mcpClient, err := client.NewInProcessClient(mcpServer) //同一进程内直接调用
		if err != nil {
			toolsErr = err
			zlog.Error("create in-process MCP client failed: " + err.Error())
			return
		}

		// 构建初始化请求，设置协议版本和客户端信息
		initRequest := mcp.InitializeRequest{}
		initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		initRequest.Params.ClientInfo = mcp.Implementation{
			Name:    "seekF-client",
			Version: "1.0.0",
		}

		// 向MCP服务器发送初始化请求
		_, err = mcpClient.Initialize(ctx, initRequest)
		if err != nil {
			toolsErr = err
			zlog.Error("initialize MCP client failed: " + err.Error())
			return
		}

		// 从MCP服务器获取可用的工具列表
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

// GetMCPTools 获取MCP工具列表，如果尚未初始化则先进行初始化
func GetMCPTools(ctx context.Context) ([]tool.BaseTool, error) {
	if len(einoTools) == 0 {
		return GetInProcessTools(ctx)
	}
	return einoTools, nil
}
