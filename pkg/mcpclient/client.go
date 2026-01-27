package mcpclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"shell-executor-mcp/pkg/configs"

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
	// 确定要连接的服务器 URL
	serverURL := c.serverURL
	if serverURL == "" {
		if len(c.config.Servers) == 0 {
			return errors.New("没有可用的服务器")
		}
		serverURL = c.config.Servers[0].URL
	}

	c.logger.Debugf("连接到服务器: %s", serverURL)

	// 创建 MCP Client
	c.client = mcp.NewClient(&mcp.Implementation{
		Name:    "shell-executor-client",
		Version: "1.0.0",
	}, nil)

	// 创建 StreamableClientTransport 用于 SSE 连接
	transport := &mcp.StreamableClientTransport{
		Endpoint:   serverURL,
		HTTPClient: c.httpClient,
	}

	// 尝试连接
	session, err := c.client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("连接服务器失败: %w", err)
	}

	c.session = session
	c.logger.Infof("成功连接到服务器: %s", serverURL)
	return nil
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	if c.session != nil {
		return c.session.Close()
	}
	return nil
}

// ExecuteCommand 执行命令
// command: 要执行的命令
// 返回执行结果
func (c *Client) ExecuteCommand(ctx context.Context, command string) (*Result, error) {
	if c.session == nil {
		return nil, errors.New("客户端未连接，请先调用 Connect()")
	}

	c.logger.Debugf("执行命令: %s", command)

	// 调用 MCP Tool: execute_command
	result, err := c.session.CallTool(ctx, &mcp.CallToolParams{
		Name: "execute_command",
		Arguments: map[string]any{
			"command": command,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("调用 MCP Tool 失败: %w", err)
	}

	// 解析结果
	return ParseResult(result), nil
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
