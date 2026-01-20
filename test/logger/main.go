package main

import (
	"fmt"
	"os"
	"path/filepath"

	"shell-executor-mcp/internal/logger"
)

// TestLogger 测试日志功能
func TestLogger() {
	// 测试日志初始化
	logCfg := logger.DefaultLogConfig()

	// 创建日志目录
	if err := os.MkdirAll(logCfg.LogDir, 0755); err != nil {
		fmt.Printf("Failed to create log directory: %v\n", err)
		os.Exit(1)
	}

	// 构建日志文件路径
	logFilePath := filepath.Join(logCfg.LogDir, "test.log")
	fmt.Printf("Log file path: %s\n", logFilePath)

	// 初始化日志
	if err := logger.InitLogger(logCfg, "test.log"); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// 测试日志输出
	logger.Info("Test log message")
	logger.Debug("Test debug message")
	logger.Warn("Test warn message")
	logger.Error("Test error message")

	// 同步日志
	logger.Sync()

	fmt.Println("Logger test completed successfully!")
}

func main() {
	TestLogger()
}
