package main

import (
	"github.com/AceDarkknight/shell-executor-mcp/internal/logger"
)

func main() {
	logger.InitLogger(nil, "simple_test.log")
	logger.Info("Simple server test - no logger")
	logger.Info("Server test completed successfully!")
}
