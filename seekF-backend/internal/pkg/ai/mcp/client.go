package mcp

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"seekF-backend/internal/pkg/zlog"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

var (
	einoTools          []einotool.BaseTool               // 存储已初始化的Eino工具列表
	allToolInfos       []*schema.ToolInfo                // 缓存的工具声明（避免每次请求调用 t.Info）
	allInvokableByName map[string]einotool.InvokableTool // 工具名 -> 可调用实现
	toolsOnce          sync.Once                         // 确保工具只初始化一次的同步原语
	toolsErr           error                             // 记录初始化过程中的错误
)

// GetInProcessTools 获取当前进程内的MCP工具实例
func GetInProcessTools(ctx context.Context) ([]einotool.BaseTool, error) {
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

		// 从MCP服务器获取可用的工具列表，eino-ext 的适配器把 MCP 工具转成 Eino 的 BaseTool 接口
		einoTools, err = mcpp.GetTools(ctx, &mcpp.Config{
			Cli: mcpClient,
		})
		if err != nil {
			toolsErr = err
			zlog.Error("get MCP tools failed: " + err.Error())
			return
		}

		// 初始化阶段用独立 context 拉取工具元信息，避免上游请求 ctx 过短导致缓存未建立
		metaCtx := context.Background()
		allToolInfos = make([]*schema.ToolInfo, 0, len(einoTools))
		allInvokableByName = make(map[string]einotool.InvokableTool)
		for _, t := range einoTools {
			info, ierr := t.Info(metaCtx)
			if ierr != nil {
				zlog.Error("tool Info failed: " + ierr.Error())
				continue
			}
			allToolInfos = append(allToolInfos, info)
			if inv, ok := t.(einotool.InvokableTool); ok {
				allInvokableByName[info.Name] = inv
			}
		}

		zlog.Info(fmt.Sprintf("MCP tools initialized: %d tools, %d tool infos cached", len(einoTools), len(allToolInfos)))
	})
	return einoTools, toolsErr
}

// GetMCPTools 获取MCP工具列表，如果尚未初始化则先进行初始化
func GetMCPTools(ctx context.Context) ([]einotool.BaseTool, error) {
	if len(einoTools) == 0 {
		return GetInProcessTools(ctx)
	}
	return einoTools, nil
}

// FilteredMCPToolBinding 返回按开关过滤后的工具声明与可调用映射（复用进程内 ToolInfo 缓存）
func FilteredMCPToolBinding(ctx context.Context, enableWebSearch bool) ([]*schema.ToolInfo, map[string]einotool.InvokableTool, error) {
	_, err := GetMCPTools(ctx)
	if err != nil {
		return nil, nil, err
	}
	if len(allToolInfos) == 0 {
		return nil, nil, nil
	}
	outInfos := make([]*schema.ToolInfo, 0, len(allToolInfos))
	outByName := make(map[string]einotool.InvokableTool)
	for _, info := range allToolInfos {
		// 根据开关过滤 web_search 工具
		if !enableWebSearch && info.Name == "web_search" {
			continue
		}
		outInfos = append(outInfos, info)
		if inv, ok := allInvokableByName[info.Name]; ok {
			outByName[info.Name] = inv
		}
	}
	if len(outInfos) == 0 {
		return nil, nil, nil
	}
	return outInfos, outByName, nil
}
