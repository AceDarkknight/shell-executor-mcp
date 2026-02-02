package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/AceDarkknight/shell-executor-mcp/pkg/mcpclient"

	"github.com/AceDarkknight/shell-executor-mcp/pkg/configs"

	"github.com/AceDarkknight/shell-executor-mcp/internal/logger"
)

// 本示例演示如何使用 mcpclient 包
func main() {
	// 1. 加载配置
	cfg, err := configs.LoadClientConfig("../client_config.json")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 2. 初始化日志
	logCfg := cfg.Log.ToLoggerConfig()
	if err := logger.InitLogger(&logCfg, "test_mcpclient.log"); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Sync()

	fmt.Println("=== MCP Client SDK 测试 ===")
	fmt.Printf("配置加载成功，服务器数量: %d\n", len(cfg.Servers))
	for _, server := range cfg.Servers {
		fmt.Printf("  - %s: %s\n", server.Name, server.URL)
	}

	// 3. 创建客户端（使用配置文件）
	fmt.Println("\n--- 测试 1: 使用配置文件创建客户端 ---")
	client1, err := mcpclient.NewClient(cfg)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	fmt.Println("客户端创建成功")

	// 4. 测试配置验证
	fmt.Println("\n--- 测试 2: 配置验证 ---")
	// 测试 nil 配置
	_, err = mcpclient.NewClient(nil)
	if err != nil {
		fmt.Printf("✓ nil 配置验证通过: %v\n", err)
	}

	// 测试空服务器列表
	emptyCfg := &configs.ClientConfig{
		Servers: []configs.ServerConfig{},
	}
	_, err = mcpclient.NewClient(emptyCfg)
	if err != nil {
		fmt.Printf("✓ 空服务器列表验证通过: %v\n", err)
	}

	// 测试无效服务器配置
	invalidCfg := &configs.ClientConfig{
		Servers: []configs.ServerConfig{
			{Name: "", URL: ""},
		},
	}
	_, err = mcpclient.NewClient(invalidCfg)
	if err != nil {
		fmt.Printf("✓ 无效服务器配置验证通过: %v\n", err)
	}

	// 5. 测试选项模式
	fmt.Println("\n--- 测试 3: 选项模式 ---")
	_, err = mcpclient.NewClient(cfg,
		mcpclient.WithTimeout(60*time.Second),
		mcpclient.WithHeader("X-Custom-Header", "test-value"),
	)
	if err != nil {
		log.Fatalf("使用选项创建客户端失败: %v", err)
	}
	fmt.Println("使用选项创建客户端成功")

	// 6. 测试自定义日志记录器
	fmt.Println("\n--- 测试 4: 自定义日志记录器 ---")
	customLogger := &CustomLogger{}
	_, err = mcpclient.NewClient(cfg,
		mcpclient.WithLogger(customLogger),
	)
	if err != nil {
		log.Fatalf("使用自定义日志创建客户端失败: %v", err)
	}
	fmt.Println("使用自定义日志创建客户端成功")

	// 7. 测试连接（需要服务器运行）
	fmt.Println("\n--- 测试 5: 连接服务器 ---")
	fmt.Println("注意: 此测试需要服务器正在运行")
	fmt.Println("如果服务器未运行，连接将失败，这是正常的")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client1.Connect(ctx)
	if err != nil {
		fmt.Printf("连接失败（预期中，如果服务器未运行）: %v\n", err)
	} else {
		fmt.Println("连接成功！")
		defer client1.Close()

		// 8. 测试执行命令
		fmt.Println("\n--- 测试 6: 执行命令 ---")
		result, err := client1.ExecuteCommand(ctx, "echo 'Hello from MCP Client'")
		if err != nil {
			fmt.Printf("执行命令失败: %v\n", err)
		} else {
			fmt.Printf("命令执行成功！\n")
			fmt.Printf("结果:\n%s\n", result.String())

			// 测试获取文本内容
			texts := result.GetTextContents()
			fmt.Printf("文本内容数量: %d\n", len(texts))

			// 测试获取聚合结果
			aggregatedResults := result.GetAggregatedResults()
			fmt.Printf("聚合结果数量: %d\n", len(aggregatedResults))
		}
	}

	fmt.Println("\n=== 测试完成 ===")
}

// CustomLogger 自定义日志记录器实现
type CustomLogger struct{}

func (l *CustomLogger) Debugf(template string, args ...interface{}) {
	fmt.Printf("[CUSTOM DEBUG] "+template+"\n", args...)
}

func (l *CustomLogger) Infof(template string, args ...interface{}) {
	fmt.Printf("[CUSTOM INFO] "+template+"\n", args...)
}

func (l *CustomLogger) Warnf(template string, args ...interface{}) {
	fmt.Printf("[CUSTOM WARN] "+template+"\n", args...)
}

func (l *CustomLogger) Errorf(template string, args ...interface{}) {
	fmt.Printf("[CUSTOM ERROR] "+template+"\n", args...)
}
