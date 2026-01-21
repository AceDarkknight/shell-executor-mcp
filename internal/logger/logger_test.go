package logger

import (
	"os"
	"sync"
	"testing"

	"go.uber.org/zap"
)

// TestConcurrentInitLogger 测试并发调用 InitLogger 的安全性
func TestConcurrentInitLogger(t *testing.T) {
	// 使用独立的日志目录
	logDir := "logs_test_concurrent_init"
	// 清理旧的测试日志目录
	_ = os.RemoveAll(logDir)

	var wg sync.WaitGroup
	errors := make(chan error, 100)
	successCount := 0
	var mu sync.Mutex

	// 启动 100 个 goroutine 同时调用 InitLogger
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cfg := &LogConfig{
				Level:      "debug",
				LogDir:     logDir,
				MaxSize:    100,
				MaxBackups: 3,
				MaxAge:     28,
				Compress:   false,
			}

			err := InitLogger(cfg, "test_concurrent.log")
			if err != nil {
				errors <- err
			} else {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// 检查是否有错误
	for err := range errors {
		t.Errorf("InitLogger failed: %v", err)
	}

	// 验证所有调用都成功
	if successCount != 100 {
		t.Errorf("Expected 100 successful calls, got %d", successCount)
	}

	// 验证 logger 已正确初始化
	if L() == nil {
		t.Error("Logger should be initialized")
	}
	if S() == nil {
		t.Error("Sugar logger should be initialized")
	}

	// 同步日志并清理测试文件
	_ = Sync()
	t.Cleanup(func() {
		_ = os.RemoveAll(logDir)
	})
}

// TestConcurrentL 测试并发调用 L() 的安全性
func TestConcurrentL(t *testing.T) {
	// 使用独立的日志目录
	logDir := "logs_test_concurrent_l"
	// 清理旧的测试日志目录
	_ = os.RemoveAll(logDir)

	var wg sync.WaitGroup

	// 启动 100 个 goroutine 同时调用 L()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger := L()
			if logger == nil {
				t.Error("L() should return non-nil logger")
			}
		}()
	}

	wg.Wait()

	// 验证 logger 已正确初始化
	if L() == nil {
		t.Error("Logger should be initialized")
	}

	// 同步日志并清理测试文件
	_ = Sync()
	t.Cleanup(func() {
		_ = os.RemoveAll(logDir)
	})
}

// TestConcurrentS 测试并发调用 S() 的安全性
func TestConcurrentS(t *testing.T) {
	// 使用独立的日志目录
	logDir := "logs_test_concurrent_s"
	// 清理旧的测试日志目录
	_ = os.RemoveAll(logDir)

	var wg sync.WaitGroup

	// 启动 100 个 goroutine 同时调用 S()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger := S()
			if logger == nil {
				t.Error("S() should return non-nil logger")
			}
		}()
	}

	wg.Wait()

	// 验证 sugar logger 已正确初始化
	if S() == nil {
		t.Error("Sugar logger should be initialized")
	}

	// 同步日志并清理测试文件
	_ = Sync()
	t.Cleanup(func() {
		_ = os.RemoveAll(logDir)
	})
}

// TestConcurrentLogging 测试并发写日志的安全性
func TestConcurrentLogging(t *testing.T) {
	// 使用独立的日志目录
	logDir := "logs_test_concurrent_logging"
	// 清理旧的测试日志目录
	_ = os.RemoveAll(logDir)

	// 初始化 logger
	cfg := &LogConfig{
		Level:      "debug",
		LogDir:     logDir,
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   false,
	}
	err := InitLogger(cfg, "test_logging.log")
	if err != nil {
		t.Fatalf("InitLogger failed: %v", err)
	}

	var wg sync.WaitGroup

	// 启动 100 个 goroutine 同时写日志
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			Info("Test message", zap.Int("index", idx))
			Debug("Debug message", zap.Int("index", idx))
			Warn("Warning message", zap.Int("index", idx))
		}(i)
	}

	wg.Wait()

	// 同步日志
	err = Sync()
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}

	// 清理测试文件
	t.Cleanup(func() {
		_ = os.RemoveAll(logDir)
	})
}

// TestConcurrentClientAndServer 模拟既是 client 又是 server 的场景
func TestConcurrentClientAndServer(t *testing.T) {
	// 使用独立的日志目录
	logDir := "logs_test_client_server"
	// 清理旧的测试日志目录
	_ = os.RemoveAll(logDir)

	var wg sync.WaitGroup
	errors := make(chan error, 2)

	// 模拟 client 初始化
	wg.Add(1)
	go func() {
		defer wg.Done()
		cfg := &LogConfig{
			Level:      "debug",
			LogDir:     logDir,
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   false,
		}
		// 注意：使用同一个文件名，通过字段区分
		err := InitLogger(cfg, "app.log")
		if err != nil {
			errors <- err
		}
	}()

	// 模拟 server 初始化
	wg.Add(1)
	go func() {
		defer wg.Done()
		cfg := &LogConfig{
			Level:      "debug",
			LogDir:     logDir,
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   false,
		}
		// 注意：使用同一个文件名，通过字段区分
		err := InitLogger(cfg, "app.log")
		if err != nil {
			errors <- err
		}
	}()

	wg.Wait()
	close(errors)

	// 检查是否有错误
	for err := range errors {
		t.Errorf("InitLogger failed: %v", err)
	}

	// 验证 logger 已正确初始化
	if L() == nil {
		t.Error("Logger should be initialized")
	}

	// 模拟 client 和 server 同时写日志
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// Client 日志，添加 role 字段
			Info("Client message", zap.String("role", "client"), zap.Int("index", idx))
		}(i)

		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// Server 日志，添加 role 字段
			Info("Server message", zap.String("role", "server"), zap.Int("index", idx))
		}(i)
	}

	wg.Wait()

	// 同步日志
	err := Sync()
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}

	// 清理测试文件
	t.Cleanup(func() {
		_ = os.RemoveAll(logDir)
	})
}

// TestMultipleInitCalls 测试多次调用 InitLogger 的行为
func TestMultipleInitCalls(t *testing.T) {
	// 使用独立的日志目录
	logDir := "logs_test_multiple_init"
	// 清理旧的测试日志目录
	_ = os.RemoveAll(logDir)

	cfg1 := &LogConfig{
		Level:      "debug",
		LogDir:     logDir,
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   false,
	}

	cfg2 := &LogConfig{
		Level:      "info",
		LogDir:     logDir,
		MaxSize:    200,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}

	// 第一次初始化
	err := InitLogger(cfg1, "test_multiple.log")
	if err != nil {
		t.Fatalf("First InitLogger failed: %v", err)
	}

	logger1 := L()

	// 第二次初始化（应该被忽略）
	err = InitLogger(cfg2, "test_multiple.log")
	if err != nil {
		t.Errorf("Second InitLogger should not fail: %v", err)
	}

	logger2 := L()

	// 验证两次返回的是同一个 logger 实例
	if logger1 != logger2 {
		t.Error("InitLogger should only initialize once, returning same logger instance")
	}

	// 同步日志并清理测试文件
	_ = Sync()
	t.Cleanup(func() {
		_ = os.RemoveAll(logDir)
	})
}
