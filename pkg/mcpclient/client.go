package mcpclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/AceDarkknight/shell-executor-mcp/pkg/configs"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Client 封装了 MCP Client 和 Session，提供高级接口
type Client struct {
	client     *mcp.Client
	session    *mcp.ClientSession
	config     *configs.ClientConfig
	logger     Logger
	httpClient *http.Client
	timeout    time.Duration
	headers    map[string]string
	serverURL  string
	// 心跳机制相关字段
	cancelHeartbeat context.CancelFunc // 用于停止心跳协程
	heartbeatCtx    context.Context    // 心跳协程的上下文
	// 重连机制相关字段
	mu           sync.Mutex    // 保护 session 状态
	isConnecting bool          // 标记是否正在进行连接/重连，防止重连风暴
	connectChan  chan struct{} // 用于连接控制
}

// NewClient 创建一个新的 MCP 客户端
// cfg: 客户端配置（必需），如果为 nil 或无效，将返回错误
// opts: 可选配置参数
func NewClient(cfg *configs.ClientConfig, opts ...Option) (*Client, error) {
	// 校验配置参数
	if cfg == nil {
		return nil, errors.New("配置参数不能为空")
	}

	if len(cfg.Servers) == 0 {
		return nil, errors.New("服务器列表不能为空")
	}

	// 验证每个服务器的配置
	for i, server := range cfg.Servers {
		if server.Name == "" {
			return nil, fmt.Errorf("服务器 [%d] 的名称不能为空", i)
		}
		if server.URL == "" {
			return nil, fmt.Errorf("服务器 [%d] 的地址不能为空", i)
		}
	}

	// 创建客户端实例
	client := &Client{
		config:     cfg,
		logger:     GetDefaultLogger(),
		timeout:    30 * time.Second,
		httpClient: &http.Client{},
		headers:    make(map[string]string),
	}

	// 应用可选参数
	for _, opt := range opts {
		opt(client)
	}

	// 如果没有设置超时，使用默认值
	if client.httpClient.Timeout == 0 {
		client.httpClient.Timeout = client.timeout
	}

	return client, nil
}

// Connect 连接到服务器
// 如果没有指定 serverURL，则尝试连接配置中的第一个可用服务器
func (c *Client) Connect(ctx context.Context) error {
	// 尝试获取连接令牌
	c.mu.Lock()
	if c.isConnecting {
		c.mu.Unlock()
		return errors.New("正在尝试连接，请稍后重试")
	}
	c.isConnecting = true
	c.mu.Unlock()

	// 使用 defer 确保状态被重置
	defer func() {
		c.mu.Lock()
		c.isConnecting = false
		c.mu.Unlock()
	}()

	// 确定要连接的服务器 URL
	c.mu.Lock()
	serverURL := c.serverURL
	if serverURL == "" {
		if len(c.config.Servers) == 0 {
			c.mu.Unlock()
			return errors.New("没有可用的服务器")
		}
		serverURL = c.config.Servers[0].URL
	}
	c.mu.Unlock()

	c.logger.Debugf("连接到服务器: %s", serverURL)

	// 创建 MCP Client
	newClient := mcp.NewClient(&mcp.Implementation{
		Name:    "shell-executor-client",
		Version: "1.0.0",
	}, nil)

	// 创建 StreamableClientTransport 用于 SSE 连接
	transport := &mcp.StreamableClientTransport{
		Endpoint:   serverURL,
		HTTPClient: c.httpClient,
	}

	session, err := newClient.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("连接服务器失败: %w", err)
	}

	// 获取锁，更新 session 和 client
	c.mu.Lock()
	c.client = newClient
	c.session = session
	c.mu.Unlock()

	c.logger.Infof("成功连接到服务器: %s", serverURL)

	// 启动心跳机制（在锁保护内调用）
	c.mu.Lock()
	c.startHeartbeatLocked()
	c.mu.Unlock()

	return nil
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	c.mu.Lock()
	// 停止心跳协程
	if c.cancelHeartbeat != nil {
		c.cancelHeartbeat()
		c.cancelHeartbeat = nil
	}
	c.heartbeatCtx = nil

	// 关闭 session
	session := c.session
	c.session = nil
	c.client = nil
	c.mu.Unlock()

	if session != nil {
		return session.Close()
	}
	return nil
}

// ExecuteCommand 执行命令
// command: 要执行的命令
// 返回执行结果
func (c *Client) ExecuteCommand(ctx context.Context, command string) (*Result, error) {
	c.mu.Lock()
	session := c.session
	c.mu.Unlock()

	if session == nil {
		return nil, errors.New("客户端未连接，请先调用 Connect()")
	}

	c.logger.Debugf("执行命令: %s", command)

	// 最大重试次数
	const maxRetries = 3

	for i := 0; i < maxRetries; i++ {
		// 调用 MCP Tool: execute_command
		result, err := c.executeTool(ctx, session, command)

		// 如果成功，直接返回结果
		if err == nil {
			return result, nil
		}

		// 检查错误是否为连接相关错误
		if !c.isConnectionError(err) {
			// 非连接错误，直接返回
			return nil, fmt.Errorf("调用 MCP Tool 失败: %w", err)
		}

		// 是连接错误，尝试重连
		c.logger.Warnf("检测到连接错误，尝试重连 (第 %d/%d 次)", i+1, maxRetries)

		// 重新建立连接
		reconnectErr := c.Connect(ctx)
		if reconnectErr != nil {
			c.logger.Errorf("重连失败: %v", reconnectErr)
			// 重连失败，等待后继续重试
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		// 重连成功，获取新的 session
		c.mu.Lock()
		session = c.session
		c.mu.Unlock()

		if session == nil {
			return nil, errors.New("重连后 session 为空")
		}
	}

	// 达到最大重试次数
	return nil, fmt.Errorf("执行命令失败，已达到最大重试次数 %d 次", maxRetries)
}

// executeTool 执行 MCP Tool 调用
func (c *Client) executeTool(ctx context.Context, session *mcp.ClientSession, command string) (*Result, error) {
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "execute_command",
		Arguments: map[string]any{
			"command": command,
		},
	})

	if err != nil {
		return nil, err
	}

	// 解析结果
	return ParseResult(result), nil
}

// isConnectionError 检查错误是否为连接相关错误
func (c *Client) isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// 检查常见连接错误
	connectionErrors := []string{
		"EOF",
		"broken pipe",
		"connection reset",
		"connection refused",
		"connection closed",
		"stream closed",
		"i/o timeout",
		"timeout",
		"context deadline exceeded",
		"context canceled",
		"session is closed",
		"no session",
		"not connected",
	}

	errLower := strings.ToLower(errStr)
	for _, pattern := range connectionErrors {
		if strings.Contains(errLower, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// GetSession 获取底层的 MCP ClientSession
// 注意：直接使用此方法会绕过封装的高级接口
func (c *Client) GetSession() *mcp.ClientSession {
	return c.session
}

// GetClient 获取底层的 MCP Client
// 注意：直接使用此方法会绕过封装的高级接口
func (c *Client) GetClient() *mcp.Client {
	return c.client
}

// GetConfig 获取客户端配置
func (c *Client) GetConfig() *configs.ClientConfig {
	return c.config
}

// startHeartbeatLocked 在持有锁的情况下启动心跳协程
// 注意：调用此方法前必须已持有 c.mu 锁
func (c *Client) startHeartbeatLocked() {
	// 如果已有心跳，先停止（幂等性）
	if c.cancelHeartbeat != nil {
		c.cancelHeartbeat()
		c.cancelHeartbeat = nil
	}
	c.heartbeatCtx = nil

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelHeartbeat = cancel
	c.heartbeatCtx = ctx

	// 启动后台协程
	go c.runHeartbeat(ctx)
}

// startHeartbeat 启动心跳协程（外部调用版本，会自动加锁）
func (c *Client) startHeartbeat() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.startHeartbeatLocked()
}

// runHeartbeat 运行心跳循环
// ctx 参数用于控制协程生命周期
func (c *Client) runHeartbeat(ctx context.Context) {
	// 使用 time.Ticker 定期发送心跳请求
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 收到取消信号，退出协程
			c.logger.Debugf("心跳协程已停止")
			return
		case <-ticker.C:
			// 发送心跳请求
			c.sendHeartbeat()
		}
	}
}

// sendHeartbeat 发送心跳请求
func (c *Client) sendHeartbeat() {
	c.mu.Lock()
	session := c.session
	c.mu.Unlock()

	if session == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := session.ListTools(ctx, nil)
	if err != nil {
		c.logger.Warnf("心跳请求失败: %v", err)
	} else {
		c.logger.Debugf("心跳请求成功")
	}
}
