package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
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
				logger.Fatalf("Failed to load config: %v", err)
			}
		} else {
			// 从 viper 读取配置（可能来自环境变量或默认配置文件）
			cfg, err = loadConfigFromViper()
			if err != nil {
				logger.Fatalf("Failed to load config from viper: %v", err)
			}
		}

		// 初始化日志
		logCfg := cfg.Log.ToLoggerConfig()
		if err := logger.InitLogger(logCfg, "client.log"); err != nil {
			logger.Fatalf("Failed to initialize logger: %v", err)
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

	// 如果提供了 server，添加到 servers 列表
	if server != "" {
		cfg.Servers = []configs.ServerConfig{
			{
				Name: "default",
				URL:  server,
			},
		}
	}

	return cfg, nil
}

// connectToAvailableServer 尝试连接服务器列表中的第一个可用服务器
func connectToAvailableServer(servers []configs.ServerConfig) (*MCPClientSession, error) {
	logger.Debugf("开始尝试连接服务器，服务器列表长度: %d", len(servers))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for i, serverCfg := range servers {
		logger.Infof("尝试连接服务器 [%d/%d]: %s (%s)", i+1, len(servers), serverCfg.Name, serverCfg.URL)

		// 创建 MCP Client
		logger.Debugf("创建 MCP Client: name=shell-executor-client, version=1.0.0")
		client := mcp.NewClient(&mcp.Implementation{
			Name:    "shell-executor-client",
			Version: "1.0.0",
		}, nil)

		// 创建 StreamableClientTransport 用于 SSE 连接
		logger.Debugf("创建 StreamableClientTransport: endpoint=%s, timeout=30s", serverCfg.URL)
		transport := &mcp.StreamableClientTransport{
			Endpoint: serverCfg.URL,
			HTTPClient: &http.Client{
				Timeout: 30 * time.Second,
			},
		}

		// 尝试连接
		logger.Debugf("开始连接到服务器...")
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			logger.Warnf("连接到服务器 %s 失败: %v", serverCfg.Name, err)
			continue
		}

		// 连接成功
		logger.Infof("成功连接到服务器: %s (%s)", serverCfg.Name, serverCfg.URL)
		return &MCPClientSession{
			Client:  client,
			Session: session,
			URL:     serverCfg.URL,
		}, nil
	}

	logger.Errorf("所有服务器连接尝试都失败，没有可用的服务器")
	return nil, errors.New("no available server found")
}

// runCLI 运行交互式命令行界面
func runCLI(clientSession *MCPClientSession) {
	logger.Debugf("启动交互式命令行界面")
	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	logger.Info("Shell Executor MCP Client")
	logger.Info("Type 'exit' or 'quit' to exit")
	logger.Info("----------------------------------------")

	for {
		logger.Info("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			logger.Errorf("读取用户输入失败: %v", err)
			break
		}

		cmd := strings.TrimSpace(input)
		logger.Debugf("用户输入: %s", cmd)

		if cmd == "exit" || cmd == "quit" {
			logger.Info("用户请求退出")
			logger.Info("Goodbye!")
			break
		}

		if cmd == "" {
			logger.Debugf("用户输入为空，跳过")
			continue
		}

		// 发送命令到 Server
		logger.Infof("准备执行命令: %s", cmd)
		// 调用 MCP Tool: execute_command
		logger.Debugf("调用 MCP Tool: execute_command，参数: command=%s", cmd)
		result, err := clientSession.Session.CallTool(ctx, &mcp.CallToolParams{
			Name: "execute_command",
			Arguments: map[string]any{
				"command": cmd,
			},
		})

		if err != nil {
			logger.Errorf("调用 MCP Tool 失败: %v", err)
			logger.Infof("----------------------------------------")
			continue
		}

		if result.IsError {
			logger.Warnf("服务器返回错误，命令: %s", cmd)
			for i, content := range result.Content {
				if text, ok := content.(*mcp.TextContent); ok {
					logger.Infof("错误内容 [%d]: %s", i, text.Text)
					logger.Infof("  %s\n", text.Text)
				}
			}
			logger.Info("----------------------------------------")
			continue
		}

		logger.Debugf("命令执行成功，开始处理结果，内容数量: %d", len(result.Content))
		// 解析并显示结果
		displayResult(result.Content)
		logger.Info("----------------------------------------")
	}
}

// displayResult 解析并显示 MCP Tool 返回的结果
func displayResult(contents []mcp.Content) {
	logger.Debugf("开始解析和显示结果，内容数量: %d", len(contents))

	for i, content := range contents {
		logger.Debugf("处理内容 [%d]: %T", i, content)
		switch v := content.(type) {
		case *mcp.TextContent:
			logger.Debugf("文本内容长度: %d", len(v.Text))
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
				logger.Infof("成功解析聚合结果，组数: %d, 摘要: %s\n", len(aggregatedResult.Groups), aggregatedResult.Summary)
				for j, group := range aggregatedResult.Groups {
					logger.Infof("显示组 [%d]: count=%d, status=%s", j+1, group.Count, group.Status)
					if group.Output != "" {
						logger.Infof("组 [%d] 输出长度: %d", j+1, len(group.Output))
						logger.Infof("Output:\n%s\n", group.Output)
					}
					if group.Error != "" {
						logger.Infof("组 [%d] 错误: %s", j+1, group.Error)
						logger.Infof("Error: %s\n", group.Error)
					}
					if len(group.Nodes) > 0 {
						logger.Debugf("组 [%d] 节点数: %d", j+1, len(group.Nodes))
						// 显示节点列表，如果太多则截断
						nodesStr := strings.Join(group.Nodes, ", ")
						if len(nodesStr) > 100 {
							nodesStr = nodesStr[:100] + "..."
						}
						logger.Infof("Nodes: %s\n", nodesStr)
					}
				}
			} else {
				// 无法解析为 JSON，直接显示文本
				logger.Infof("无法解析为 JSON，作为纯文本显示，解析错误: %v", err)
				logger.Infof("%s\n", v.Text)
			}
		default:
			logger.Infof("未知的内容类型: %T", content)
		}
	}
	logger.Infof("结果显示完成")
}
