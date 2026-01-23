package main

import (
	"os"

	"shell-executor-mcp/internal/logger"
)

// TestLogger 测试日志功能
func TestLogger() {
	// 测试日志初始化
	logCfg := logger.DefaultLogConfig()

	// 创建日志目录
	if err := os.MkdirAll(logCfg.LogDir, 0755); err != nil {
		logger.Fatalf("Failed to create log directory: %v", err)
	}

	// 初始化日志
	if err := logger.InitLogger(logCfg, "test.log"); err != nil {
		logger.Fatalf("Failed to initialize logger: %v", err)
	}

	// 测试日志输出
	logger.Info("Test log message")
	logger.Debug("Test debug message")
	logger.Warn("Test warn message")
	logger.Error("Test error message")

	// 同步日志
	logger.Sync()

	logger.Info("Logger test completed successfully!")
}

func main() {
	TestLogger()
}
