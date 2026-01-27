package mcpclient

import (
	"net/http"
	"time"
)

// Option 定义客户端的可选配置参数
type Option func(*Client)

// WithLogger 设置自定义日志记录器
func WithLogger(l Logger) Option {
	return func(c *Client) {
		c.logger = l
	}
}

// WithTimeout 设置连接和请求超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithHTTPClient 设置自定义 HTTP 客户端
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithHeaders 设置请求头
func WithHeaders(headers map[string]string) Option {
	return func(c *Client) {
		c.headers = headers
	}
}

// WithHeader 添加单个请求头
func WithHeader(key, value string) Option {
	return func(c *Client) {
		if c.headers == nil {
			c.headers = make(map[string]string)
		}
		c.headers[key] = value
	}
}

// WithServerURL 设置服务器地址（覆盖配置中的服务器列表）
func WithServerURL(url string) Option {
	return func(c *Client) {
		c.serverURL = url
	}
}
