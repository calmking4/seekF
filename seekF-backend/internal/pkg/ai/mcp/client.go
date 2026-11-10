package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/zlog"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

var (
	// 保护退出时关闭客户端与请求读取工具缓存的并发访问。
	clientMu           sync.RWMutex
	stdioClient        *client.Client
	einoTools          []einotool.BaseTool
	allToolInfos       []*schema.ToolInfo
	allInvokableByName map[string]einotool.InvokableTool
	toolsErr           = errors.New("MCP客户端未初始化")
)

// InitMCPClient 显式启动stdio子进程并加载工具，使用有限重试且只发布完整的工具缓存。
func InitMCPClient(ctx context.Context) error {
	clientMu.Lock()
	defer clientMu.Unlock()
	if stdioClient != nil {
		return nil
	}
	var lastErr error
	for attempt := 1; attempt <= constants.MCPInitAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		candidate, tools, infos, invokable, err := loadMCPClient(ctx)
		if err == nil {
			stdioClient, einoTools, allToolInfos, allInvokableByName = candidate, tools, infos, invokable
			toolsErr = nil
			zlog.Info(fmt.Sprintf("MCP客户端初始化完成，工具数=%d", len(tools)))
			return nil
		}
		lastErr = err
		toolsErr = err
		zlog.Warn(fmt.Sprintf("MCP客户端初始化失败，第%d次尝试: %s", attempt, err.Error()))
		if attempt < constants.MCPInitAttempts {
			timer := time.NewTimer(constants.MCPInitRetryDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return fmt.Errorf("初始化MCP客户端失败: %w", lastErr)
}

// GetMCPTools 返回已初始化工具的列表副本，不启动子进程或执行网络请求。
func GetMCPTools(ctx context.Context) ([]einotool.BaseTool, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	clientMu.RLock()
	defer clientMu.RUnlock()
	if toolsErr != nil {
		return nil, toolsErr
	}
	return append([]einotool.BaseTool(nil), einoTools...), nil
}

// CloseMCPClient 关闭stdio客户端及其子进程，清空工具缓存并允许后续显式初始化。
func CloseMCPClient() error {
	clientMu.Lock()
	defer clientMu.Unlock()
	candidate := stdioClient
	stdioClient, einoTools, allToolInfos, allInvokableByName = nil, nil, nil, nil
	toolsErr = errors.New("MCP客户端未初始化或已关闭")
	if candidate == nil {
		return nil
	}
	if err := candidate.Close(); err != nil {
		return fmt.Errorf("关闭MCP客户端失败: %w", err)
	}
	return nil
}

// FilteredMCPToolBinding 从已加载的缓存生成本次请求的工具绑定。
func FilteredMCPToolBinding(ctx context.Context, enableWebSearch bool) ([]*schema.ToolInfo, map[string]einotool.InvokableTool, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	clientMu.RLock()
	defer clientMu.RUnlock()
	if toolsErr != nil {
		return nil, nil, toolsErr
	}
	infos := make([]*schema.ToolInfo, 0, len(allToolInfos))
	invokable := make(map[string]einotool.InvokableTool)
	for _, info := range allToolInfos {
		if !enableWebSearch && info.Name == "web_search" {
			continue
		}
		infos = append(infos, info)
		if tool, ok := allInvokableByName[info.Name]; ok {
			invokable[info.Name] = tool
		}
	}
	return infos, invokable, nil
}

// loadMCPClient 单次创建子进程，以独立的限时上下文完成握手和工具加载，失败时清理资源。
func loadMCPClient(lifetimeCtx context.Context) (_ *client.Client, _ []einotool.BaseTool, _ []*schema.ToolInfo, _ map[string]einotool.InvokableTool, resultErr error) {
	command := strings.TrimSpace(os.Getenv("MCP_SERVER_COMMAND"))
	if command == "" {
		command = "go run ./cmd/mcp-server"
	}
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, nil, nil, nil, fmt.Errorf("MCP启动命令不能为空")
	}
	workDir := strings.TrimSpace(os.Getenv("MCP_SERVER_WORKDIR"))
	if workDir == "" {
		workDir = "."
	}
	stdio := transport.NewStdioWithOptions(parts[0], nil, parts[1:],
		transport.WithCommandFunc(func(ctx context.Context, name string, env, args []string) (*exec.Cmd, error) {
			cmd := exec.CommandContext(ctx, name, args...)
			cmd.Dir = workDir
			cmd.Env = append(os.Environ(), env...)
			return cmd, nil
		}),
	)
	candidate := client.NewClient(stdio)
	// 生命周期上下文不随握手完成而取消，避免初始化成功后误杀子进程。
	defer func() {
		if resultErr != nil {
			if err := candidate.Close(); err != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("清理失败的MCP客户端失败: %w", err))
			}
		}
	}()
	if err := candidate.Start(lifetimeCtx); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("启动MCP子进程失败: %w", err)
	}
	// 持续消费子进程日志，避免stderr管道写满后阻塞协议处理。
	if stderr, ok := client.GetStderr(candidate); ok {
		go func() { _, _ = io.Copy(os.Stderr, stderr) }()
	}
	ctx, cancel := context.WithTimeout(lifetimeCtx, constants.MCPInitTimeout)
	defer cancel()
	request := mcp.InitializeRequest{}
	request.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	request.Params.ClientInfo = mcp.Implementation{Name: "seekF-client", Version: "1.0.0"}
	if _, err := candidate.Initialize(ctx, request); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("MCP协议握手失败: %w", err)
	}
	tools, err := mcpp.GetTools(ctx, &mcpp.Config{Cli: candidate})
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("加载MCP工具失败: %w", err)
	}
	if len(tools) == 0 {
		return nil, nil, nil, nil, fmt.Errorf("MCP服务未返回工具")
	}
	infos := make([]*schema.ToolInfo, 0, len(tools))
	invokable := make(map[string]einotool.InvokableTool)
	for _, tool := range tools {
		info, err := tool.Info(ctx)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("加载MCP工具声明失败: %w", err)
		}
		if info == nil {
			return nil, nil, nil, nil, fmt.Errorf("MCP工具声明为空")
		}
		implementation, ok := tool.(einotool.InvokableTool)
		if !ok {
			return nil, nil, nil, nil, fmt.Errorf("MCP工具不可调用: %s", info.Name)
		}
		infos = append(infos, info)
		invokable[info.Name] = implementation
	}
	return candidate, tools, infos, invokable, nil
}
