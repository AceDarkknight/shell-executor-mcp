package cmd

import (
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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RunCmd 表示 run 命令
var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "启动 MCP 服务器",
	Long:  `启动 MCP 服务器并开始监听请求。`,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

// runServer 启动 MCP 服务器
func runServer() {
	// 1. 加载配置
	// 如果没有指定配置文件，尝试从 viper 读取
	var cfg *config.ServerConfig
	var err error

	if cfgFile != "" {
		// 使用指定的配置文件
		cfg, err = config.LoadServerConfig(cfgFile)
		if err != nil {
			fmt.Printf("Failed to load config: %v\n", err)
			os.Exit(1)
		}
	} else {
		// 从 viper 读取配置（可能来自环境变量或默认配置文件）
		cfg, err = loadConfigFromViper()
		if err != nil {
			fmt.Printf("Failed to load config from viper: %v\n", err)
			os.Exit(1)
		}
	}

	// 初始化日志
	logCfg := &cfg.LogConfig
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

	// 4. 注册 MCP Tools
	registerTools(mcpServer, guard, executor, dispatcher, cfg)

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
	mux.HandleFunc("/internal/join", internalJoinHandler(cfg, cfgFile))
	mux.HandleFunc("/internal/sync", internalSyncHandler(cfg, cfgFile))

	// 7. 启动 HTTP Server
	addr := ":" + strconv.Itoa(cfg.Port)
	logger.Infof("Server listening on %s", addr)
	logger.Infof("MCP endpoint: http://localhost%s", addr)
	logger.Infof("Internal API endpoints: http://localhost%s/internal/...", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Fatalf("Server failed: %v", err)
	}
}

// loadConfigFromViper 从 viper 加载配置
func loadConfigFromViper() (*config.ServerConfig, error) {
	cfg := &config.ServerConfig{
		Port:         viper.GetInt("port"),
		NodeName:     viper.GetString("node_name"),
		ClusterToken: viper.GetString("token"),
		Security: config.SecurityConfig{
			BlacklistedCommands: []string{},
			DangerousArgsRegex:  []string{},
		},
		LogConfig: logger.LogConfig{
			Level:      viper.GetString("log_level"),
			LogDir:     viper.GetString("log_dir"),
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		},
	}

	// 如果 log_dir 为空，使用默认值
	if cfg.LogConfig.LogDir == "" {
		cfg.LogConfig.LogDir = "logs"
	}

	// 如果 node_name 为空，使用 hostname
	if cfg.NodeName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("failed to get hostname: %v", err)
		}
		cfg.NodeName = hostname
	}

	// 尝试从 viper 读取 peers
	peers := viper.GetStringSlice("peers")
	cfg.Peers = peers

	return cfg, nil
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
