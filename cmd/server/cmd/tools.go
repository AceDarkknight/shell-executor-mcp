package cmd

import (
	"context"
	"fmt"

	"shell-executor-mcp/internal/config"
	"shell-executor-mcp/internal/dispatch"
	"shell-executor-mcp/internal/executor"
	"shell-executor-mcp/internal/logger"
	"shell-executor-mcp/internal/security"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerTools 注册所有 MCP Tools
// 将 tool 注册逻辑集中管理，便于后续添加新的 tool
func registerTools(
	mcpServer *mcp.Server,
	guard *security.Guard,
	executor *executor.Executor,
	dispatcher *dispatch.Dispatcher,
	cfg *config.ServerConfig,
) {
	// 注册 execute_command tool
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "execute_command",
		Description: "Execute a shell command on the cluster",
	}, handleExecuteCommand(guard, executor, dispatcher, cfg))

	// 在此处添加更多 tools...
	// 示例：
	// mcp.AddTool(mcpServer, &mcp.Tool{
	//     Name:        "another_tool",
	//     Description: "Another tool description",
	// }, handlerFunc)
}

// handleExecuteCommand 处理 execute_command tool 的请求
func handleExecuteCommand(
	guard *security.Guard,
	executor *executor.Executor,
	dispatcher *dispatch.Dispatcher,
	cfg *config.ServerConfig,
) func(ctx context.Context, req *mcp.CallToolRequest, input struct {
	Command string `json:"command"`
}) (*mcp.CallToolResult, struct {
	Summary string                     `json:"summary"`
	Groups  []dispatch.AggregatedGroup `json:"groups"`
}, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input struct {
		Command string `json:"command"`
	}) (*mcp.CallToolResult, struct {
		Summary string                     `json:"summary"`
		Groups  []dispatch.AggregatedGroup `json:"groups"`
	}, error) {
		logger.Debugf("Received execute_command request: %s", input.Command)

		// 1. 安全检查
		if err := guard.CheckCommand(input.Command); err != nil {
			logger.Warnf("Security violation for command: %s, error: %v", input.Command, err)
			return nil, struct {
				Summary string                     `json:"summary"`
				Groups  []dispatch.AggregatedGroup `json:"groups"`
			}{
				Summary: "Security violation",
				Groups:  []dispatch.AggregatedGroup{},
			}, fmt.Errorf("security violation: %v", err)
		}

		// 2. 分发执行 (本地 + 集群)
		logger.Infof("Dispatching command to cluster: %s", input.Command)
		groups, summary := dispatcher.Dispatch(executor, cfg.NodeName, input.Command)
		logger.Infof("Command execution completed: %s", summary)

		return nil, struct {
			Summary string                     `json:"summary"`
			Groups  []dispatch.AggregatedGroup `json:"groups"`
		}{
			Summary: summary,
			Groups:  groups,
		}, nil
	}
}
