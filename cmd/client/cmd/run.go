package cmd

import (
	"bufio"
	"context"
	"io"
	"os"
	"strings"

	"github.com/AceDarkknight/shell-executor-mcp/pkg/mcpclient"

	"github.com/AceDarkknight/shell-executor-mcp/pkg/configs"

	"github.com/AceDarkknight/shell-executor-mcp/internal/logger"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
		if err := logger.InitLogger(&logCfg, "client.log"); err != nil {
			logger.Fatalf("Failed to initialize logger: %v", err)
		}
		defer logger.Sync()

		if len(cfg.Servers) == 0 {
			logger.Fatal("No servers configured")
		}

		logger.Infof("Client started with %d servers configured", len(cfg.Servers))

		// 2. 创建并连接客户端
		client, err := createAndConnectClient(cfg)
		if err != nil {
			logger.Fatalf("Failed to connect to server: %v", err)
		}
		defer client.Close()

		// 3. 启动交互式 CLI
		runCLI(client)
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

// createAndConnectClient 创建并连接客户端
func createAndConnectClient(cfg *configs.ClientConfig) (*mcpclient.Client, error) {
	// 准备可选参数
	var opts []mcpclient.Option

	// 如果配置中包含 token，添加到请求头
	if cfg.Token != "" {
		logger.Infof("使用 Token 认证")
		opts = append(opts, mcpclient.WithHeader("X-Cluster-Token", cfg.Token))
	}

	// 创建客户端
	client, err := mcpclient.NewClient(cfg, opts...)
	if err != nil {
		return nil, err
	}

	// 连接到服务器
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		return nil, err
	}

	return client, nil
}

// runCLI 运行交互式命令行界面
func runCLI(client *mcpclient.Client) {
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
			if err == io.EOF {
				logger.Warnf("读取用户输入结束 (EOF)")
			} else {
				logger.Errorf("读取用户输入失败: %v", err)
			}
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

		// 客户端安全检查：在发送命令到 Server 前进行拦截
		logger.Infof("准备执行命令: %s", cmd)
		result, err := client.ExecuteCommand(ctx, cmd)

		if err != nil {
			logger.Errorf("执行命令失败: %v", err)
			logger.Infof("----------------------------------------")
			continue
		}

		if result.IsError {
			logger.Warnf("服务器返回错误，命令: %s", cmd)
			logger.Infof("错误内容:\n%s", result.String())
			logger.Info("----------------------------------------")
			continue
		}

		logger.Debugf("命令执行成功，开始处理结果")
		// 显示结果
		logger.Infof("执行结果:\n%s", result.String())
		logger.Info("----------------------------------------")
	}
}
