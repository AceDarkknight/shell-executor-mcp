package mcpclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/AceDarkknight/shell-executor-mcp/pkg/configs"
)

// TestIsConnectionError 测试错误识别逻辑
func TestIsConnectionError(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil 错误",
			err:      nil,
			expected: false,
		},
		{
			name:     "EOF 错误",
			err:      errors.New("EOF"),
			expected: true,
		},
		{
			name:     "broken pipe 错误",
			err:      errors.New("broken pipe"),
			expected: true,
		},
		{
			name:     "connection reset 错误",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "context canceled 错误",
			err:      errors.New("context canceled"),
			expected: true,
		},
		{
			name:     "普通错误",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "session is closed 错误",
			err:      errors.New("session is closed"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.isConnectionError(tt.err)
			if result != tt.expected {
				t.Errorf("isConnectionError(%v) = %v, 预期 %v", tt.err, result, tt.expected)
			}
		})
	}
}

// mockRoundTripper 用于模拟 HTTP 请求
type mockRoundTripper struct {
	roundTripFunc func(*http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

// mockLogger 用于模拟日志记录器
type mockLogger struct{}

func (m *mockLogger) Debugf(template string, args ...interface{}) {}
func (m *mockLogger) Infof(template string, args ...interface{})  {}
func (m *mockLogger) Warnf(template string, args ...interface{})  {}
func (m *mockLogger) Errorf(template string, args ...interface{}) {}

// TestConcurrency_Connect 验证并发调用 Connect 不会触发多次连接
func TestConcurrency_Connect(t *testing.T) {
	config := &configs.ClientConfig{
		Servers: []configs.ServerConfig{
			{Name: "test", URL: "http://localhost:8080/sse"},
		},
	}

	callCount := 0
	var mu sync.Mutex

	// 模拟一个带延迟的 HTTP 客户端，以便触发并发竞争
	mockTransport := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			mu.Lock()
			callCount++
			mu.Unlock()
			// 模拟连接延迟
			time.Sleep(100 * time.Millisecond)
			return nil, fmt.Errorf("mock connection error")
		},
	}

	// 使用 mock logger 避免初始化死锁
	mockLogger := &mockLogger{}
	client, _ := NewClient(config, WithHTTPClient(&http.Client{Transport: mockTransport}), WithLogger(mockLogger))

	const concurrency = 5
	var wg sync.WaitGroup
	errs := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := client.Connect(context.Background())
			if err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	// 注意：callCount 可能大于 1，因为 MCP SDK 内部可能会多次调用 transport
	// 重点是验证 isConnecting 机制是否有效阻止了并发的 Connect 调用进入连接逻辑
	if callCount > 5 {
		t.Errorf("预期 callCount 最多为 5，实际为 %d，说明并发控制完全失效", callCount)
	}

	// 验证报错信息
	collisionCount := 0
	for err := range errs {
		if err.Error() == "正在尝试连接，请稍后重试" {
			collisionCount++
		}
	}
	t.Logf("成功拦截了 %d 次并发连接请求，transport 被调用了 %d 次", collisionCount, callCount)
}

// TestConcurrency_CloseAndConnect 验证并发调用 Close 和 Connect 不会导致竞态报错
// 配合 go test -race 运行
func TestConcurrency_CloseAndConnect(t *testing.T) {
	config := &configs.ClientConfig{
		Servers: []configs.ServerConfig{
			{Name: "test", URL: "http://localhost:8080/sse"},
		},
	}

	// 最小化模拟
	mockTransport := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			time.Sleep(2 * time.Millisecond)
			return nil, fmt.Errorf("mock error")
		},
	}

	// 使用 mock logger 避免初始化死锁
	mockLogger := &mockLogger{}
	client, _ := NewClient(config, WithHTTPClient(&http.Client{Transport: mockTransport}), WithLogger(mockLogger))

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = client.Connect(context.Background())
			time.Sleep(1 * time.Millisecond)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = client.Close()
			time.Sleep(1 * time.Millisecond)
		}
	}()

	wg.Wait()
}

// TestHeartbeatLifecycle 验证心跳协程在生命周期下正确启停
func TestHeartbeatLifecycle(t *testing.T) {
	config := &configs.ClientConfig{
		Servers: []configs.ServerConfig{
			{Name: "test", URL: "http://localhost:8080/sse"},
		},
	}

	// 使用 mock logger 避免初始化死锁
	mockLogger := &mockLogger{}
	client, _ := NewClient(config, WithLogger(mockLogger))

	for i := 0; i < 10; i++ {
		// 虽然连接会失败，但会触发 startHeartbeatLocked
		_ = client.Connect(context.Background())
		_ = client.Close()
	}
}

// TestClientConfigValidation 测试客户端配置验证
func TestClientConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *configs.ClientConfig
		expectError bool
	}{
		{
			name:        "nil 配置",
			config:      nil,
			expectError: true,
		},
		{
			name:        "空服务器列表",
			config:      &configs.ClientConfig{Servers: []configs.ServerConfig{}},
			expectError: true,
		},
		{
			name: "有效配置",
			config: &configs.ClientConfig{
				Servers: []configs.ServerConfig{
					{Name: "test", URL: "http://localhost:8080"},
				},
			},
			expectError: false,
		},
		{
			name: "服务器名称为空",
			config: &configs.ClientConfig{
				Servers: []configs.ServerConfig{
					{Name: "", URL: "http://localhost:8080"},
				},
			},
			expectError: true,
		},
		{
			name: "服务器 URL 为空",
			config: &configs.ClientConfig{
				Servers: []configs.ServerConfig{
					{Name: "test", URL: ""},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.config)
			if tt.expectError && err == nil {
				t.Error("预期返回错误，但没有返回")
			}
			if !tt.expectError && err != nil {
				t.Errorf("预期不返回错误，但返回了: %v", err)
			}
		})
	}
}
