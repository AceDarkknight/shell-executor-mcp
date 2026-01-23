package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"shell-executor-mcp/internal/logger"
	"shell-executor-mcp/pkg/configs"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// MCPClientSession 封装了 MCP Client 和 Session
type MCPClientSession struct {
	Client  *mcp.Client
	Session *mcp.ClientSession
	URL     string
}

// RunCmd 表示 run 命令
var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the MCP client",
	Long:  `Start the MCP client and connect to the server.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 1. 加载配置
		// 如果没有指定配置文件，尝试从 viper 读取
		var cfg *configs.ClientConfig
		var err error

		if cfgFile != "" {
			// 使用指定的配置文件
			cfg, err = configs.LoadClientConfig(cfgFile)
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
		logCfg := cfg.Log.ToLoggerConfig()
		if err := logger.InitLogger(logCfg, "client.log"); err != nil {
			fmt.Printf("Failed to initialize logger: %v\n", err)
			os.Exit(1)
		}
		defer logger.Sync()

		if len(cfg.Servers) == 0 {
			logger.Fatal("No servers configured")
		}

		logger.Infof("Client started with %d servers configured", len(cfg.Servers))

		// 2. 连接到第一个可用的 Server
		clientSession, err := connectToAvailableServer(cfg.Servers)
		if err != nil {
			logger.Fatalf("Failed to connect to any server: %v", err)
		}
		defer clientSession.Session.Close()

		logger.Infof("Connected to server: %s", clientSession.URL)

		// 3. 启动交互式 CLI
		runCLI(clientSession)
	},
}

// loadConfigFromViper 从 viper 加载配置
func loadConfigFromViper() (*configs.ClientConfig, error) {
	cfg := &configs.ClientConfig{
		Log: configs.LogConfig{
			Level:      viper.GetString("log_level"),
			LogDir:     viper.GetString("log_dir"),
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		},
	}

	// 如果 log_dir 为空，使用默认值
	if cfg.Log.LogDir == "" {
		cfg.Log.LogDir = "logs"
	}

	// 尝试从 viper 读取 server 和 token
	server := viper.GetString("server")
	token := viper.GetString("token")
	insecureSkipVerify := viper.GetBool("insecure_skip_verify")

	// 如果提供了 server，添加到 servers 列表
	if server != "" {
		cfg.Servers = []configs.ServerConfig{
			{
				Name: "default",
				URL:  server,
			},
		}
	}

	// 如果提供了 token，设置到第一个 server
	if token != "" && len(cfg.Servers) > 0 {
		// 注意：ServerConfig 结构体中没有 Token 字段
		// 这里需要根据实际情况调整
		// 暂时跳过
	}

	// 如果提供了 insecure-skip-verify，设置到第一个 server
	if insecureSkipVerify && len(cfg.Servers) > 0 {
		// 注意：ServerConfig 结构体中没有 InsecureSkipVerify 字段
		// 这里需要根据实际情况调整
		// 暂时跳过
	}

	return cfg, nil
}

// connectToAvailableServer 尝试连接服务器列表中的第一个可用服务器
func connectToAvailableServer(servers []configs.ServerConfig) (*MCPClientSession, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, serverCfg := range servers {
		logger.Infof("Trying to connect to %s (%s)...", serverCfg.Name, serverCfg.URL)

		// 创建 MCP Client
		client := mcp.NewClient(&mcp.Implementation{
			Name:    "shell-executor-client",
			Version: "1.0.0",
		}, nil)

		// 创建 StreamableClientTransport 用于 SSE 连接
		transport := &mcp.StreamableClientTransport{
			Endpoint: serverCfg.URL,
			HTTPClient: &http.Client{
				Timeout: 30 * time.Second,
			},
		}

		// 尝试连接
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			logger.Warnf("Connection failed to %s: %v", serverCfg.Name, err)
			continue
		}

		// 连接成功
		return &MCPClientSession{
			Client:  client,
			Session: session,
			URL:     serverCfg.URL,
		}, nil
	}

	return nil, fmt.Errorf("no available server found")
}

// runCLI 运行交互式命令行界面
func runCLI(clientSession *MCPClientSession) {
	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	fmt.Println("Shell Executor MCP Client")
	fmt.Println("Type 'exit' or 'quit' to exit")
	fmt.Println("----------------------------------------")

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			logger.Errorf("Error reading input: %v", err)
			break
		}

		cmd := strings.TrimSpace(input)

		if cmd == "exit" || cmd == "quit" {
			logger.Info("User requested to exit")
			fmt.Println("Goodbye!")
			break
		}

		if cmd == "" {
			continue
		}

		// 发送命令到 Server
		logger.Infof("Executing command: %s", cmd)
		fmt.Printf("Executing: %s\n", cmd)

		// 调用 MCP Tool: execute_command
		result, err := clientSession.Session.CallTool(ctx, &mcp.CallToolParams{
			Name: "execute_command",
			Arguments: map[string]any{
				"command": cmd,
			},
		})

		if err != nil {
			logger.Errorf("Error calling tool: %v", err)
			fmt.Printf("Error calling tool: %v\n", err)
			fmt.Println("----------------------------------------")
			continue
		}

		if result.IsError {
			logger.Warnf("Server returned an error for command: %s", cmd)
			fmt.Println("Server returned an error:")
			for _, content := range result.Content {
				if text, ok := content.(*mcp.TextContent); ok {
					fmt.Printf("  %s\n", text.Text)
				}
			}
			fmt.Println("----------------------------------------")
			continue
		}

		logger.Debugf("Command executed successfully, processing results")
		// 解析并显示结果
		displayResult(result.Content)
		fmt.Println("----------------------------------------")
	}
}

// displayResult 解析并显示 MCP Tool 返回的结果
func displayResult(contents []mcp.Content) {
	for _, content := range contents {
		switch v := content.(type) {
		case *mcp.TextContent:
			// 尝试解析为 JSON 格式的聚合结果
			var aggregatedResult struct {
				Summary string `json:"summary"`
				Groups  []struct {
					Count  int      `json:"count"`
					Status string   `json:"status"`
					Output string   `json:"output"`
					Error  string   `json:"error"`
					Nodes  []string `json:"nodes"`
				} `json:"groups"`
			}

			if err := json.Unmarshal([]byte(v.Text), &aggregatedResult); err == nil {
				// 成功解析为聚合结果格式
				logger.Debugf("Parsed aggregated result with %d groups", len(aggregatedResult.Groups))
				fmt.Printf("Summary: %s\n", aggregatedResult.Summary)
				fmt.Println()
				for i, group := range aggregatedResult.Groups {
					fmt.Printf("[Group %d] Count: %d | Status: %s\n", i+1, group.Count, group.Status)
					if group.Output != "" {
						fmt.Printf("Output:\n%s\n", group.Output)
					}
					if group.Error != "" {
						fmt.Printf("Error: %s\n", group.Error)
					}
					if len(group.Nodes) > 0 {
						// 显示节点列表，如果太多则截断
						nodesStr := strings.Join(group.Nodes, ", ")
						if len(nodesStr) > 100 {
							nodesStr = nodesStr[:100] + "..."
						}
						fmt.Printf("Nodes: %s\n", nodesStr)
					}
					fmt.Println()
				}
			} else {
				// 无法解析为 JSON，直接显示文本
				logger.Debugf("Could not parse result as JSON, displaying as text")
				fmt.Printf("%s\n", v.Text)
			}
		}
	}
}
