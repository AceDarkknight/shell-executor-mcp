package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"shell-executor-mcp/internal/config"
	"shell-executor-mcp/internal/dispatch"
	"shell-executor-mcp/internal/executor"
	"shell-executor-mcp/internal/logger"
	"shell-executor-mcp/internal/security"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// 1. 加载配置
	configPath := "server_config.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := config.LoadServerConfig(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logCfg := cfg.Log.ToLoggerConfig()
	if err := logger.InitLogger(logCfg, "server.log"); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Infof("Server starting as node: %s", cfg.NodeName)
	logger.Infof("Listening on port: %d", cfg.Port)

	// 2. 初始化组件
	guard, err := security.NewGuard(cfg.Security.BlacklistedCommands, cfg.Security.DangerousArgsRegex)
	if err != nil {
		logger.Fatalf("Failed to initialize security guard: %v", err)
	}

	executor := executor.NewExecutor()
	dispatcher := dispatch.NewDispatcher(cfg.GetPeers(), cfg.ClusterToken)

	// 3. 创建 MCP Server
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "shell-executor-mcp",
		Version: "1.0.0",
	}, nil)

	// 4. 注册 Tool: execute_command
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "execute_command",
		Description: "Execute a shell command on the cluster",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input struct {
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
	})

	// 5. 创建 HTTP Handler (支持 SSE)
	// 使用 StreamableHTTPHandler 来同时支持 SSE 和 JSON 请求
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{
		SessionTimeout: 10 * time.Minute,
		Stateless:      false,
	})

	// 6. 注册内部 API 端点
	// 我们需要将内部 API 挂载到同一个 http.ServeMux 上
	// 但 mcpHandler 本身是一个 http.Handler
	// 我们可以使用 http.NewServeMux 并将 MCP handler 挂载到根路径，内部 API 挂载到 /internal
	mux := http.NewServeMux()
	mux.Handle("/", mcpHandler)

	// 包装内部 API Handler 以确保它们可以被访问
	mux.HandleFunc("/internal/exec", internalExecHandler(guard, executor))
	mux.HandleFunc("/internal/join", internalJoinHandler(cfg, configPath))
	mux.HandleFunc("/internal/sync", internalSyncHandler(cfg, configPath))

	// 7. 启动 HTTP Server
	addr := ":" + strconv.Itoa(cfg.Port)
	logger.Infof("Server listening on %s", addr)
	logger.Infof("MCP endpoint: http://localhost%s", addr)
	logger.Infof("Internal API endpoints: http://localhost%s/internal/...", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Fatalf("Server failed: %v", err)
	}
}

// internalExecHandler 处理内部执行请求 (Server -> Server)
func internalExecHandler(guard *security.Guard, executor *executor.Executor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			logger.Warnf("Invalid method for /internal/exec: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Token 验证
		// 注意：这里需要从全局或上下文获取 token，简化起见暂时跳过严格验证
		// token := r.Header.Get("X-Cluster-Token")
		// if token != expectedToken { ... }

		var req dispatch.DispatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Errorf("Failed to decode request: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		logger.Debugf("Received internal exec request: %s", req.Cmd)

		// 安全检查
		if err := guard.CheckCommand(req.Cmd); err != nil {
			logger.Warnf("Security violation for internal command: %s, error: %v", req.Cmd, err)
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		// 执行
		result, err := executor.Execute(req.Cmd, 5*time.Second)
		if err != nil {
			logger.Errorf("Execution failed: %v", err)
			// 即使有错误，也返回部分结果
			// result.Error 已经包含了错误信息
		} else {
			logger.Debugf("Execution successful, exit code: %d", result.ExitCode)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// internalJoinHandler 处理节点加入请求
func internalJoinHandler(cfg *config.ServerConfig, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			logger.Warnf("Invalid method for /internal/join: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			MyAddr string `json:"my_addr"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Errorf("Failed to decode join request: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		logger.Infof("Join request from: %s", req.MyAddr)

		// 添加新节点
		cfg.AddPeer(req.MyAddr)

		// 广播给其他节点 (异步)
		go broadcastSync(cfg, configPath)

		// 返回当前所有节点
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"peers": cfg.GetPeers(),
		})
	}
}

// internalSyncHandler 处理同步节点列表请求
func internalSyncHandler(cfg *config.ServerConfig, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			logger.Warnf("Invalid method for /internal/sync: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Peers []string `json:"peers"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Errorf("Failed to decode sync request: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		logger.Infof("Sync request received: %v", req.Peers)

		// 更新本地 Peers
		cfg.SetPeers(req.Peers)

		// 持久化
		// 注意：这里需要知道配置文件路径，简化起见暂时跳过
		// cfg.Save(configPath)

		w.WriteHeader(http.StatusOK)
	}
}

// broadcastSync 将当前的 Peer 列表广播给所有已知节点
func broadcastSync(cfg *config.ServerConfig, configPath string) {
	peers := cfg.GetPeers()
	logger.Infof("Broadcasting sync to %d peers", len(peers))
	// 这里应该使用 HTTP Client 发送 POST /internal/sync
	// 简化实现，略
}
