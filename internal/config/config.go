package config

import (
	"encoding/json"
	"os"
	"slices"
	"sync"

	"shell-executor-mcp/internal/logger"
)

// ServerConfig 定义服务器的配置结构
type ServerConfig struct {
	Port         int              `json:"port"`          // 监听端口
	NodeName     string           `json:"node_name"`     // 节点名称
	Peers        []string         `json:"peers"`         // 集群中其他节点的地址列表
	Security     SecurityConfig   `json:"security"`      // 安全配置
	ClusterToken string           `json:"cluster_token"` // 集群内部通信Token
	LogConfig    logger.LogConfig `json:"log_config"`    // 日志配置
	mu           sync.RWMutex     // 读写锁，用于保护 Peers 的并发修改
}

// SecurityConfig 定义安全相关的配置
type SecurityConfig struct {
	BlacklistedCommands []string `json:"blacklisted_commands"` // 黑名单命令
	DangerousArgsRegex  []string `json:"dangerous_args_regex"` // 危险参数正则表达式
}

// LogConfig 定义日志相关的配置
type LogConfig struct {
	Level      string `json:"level"`       // 日志级别: debug, info, warn, error
	LogDir     string `json:"log_dir"`     // 日志文件目录
	MaxSize    int    `json:"max_size"`    // 单个日志文件最大大小（MB）
	MaxBackups int    `json:"max_backups"` // 保留的旧日志文件最大数量
	MaxAge     int    `json:"max_age"`     // 保留旧日志文件的最大天数
	Compress   bool   `json:"compress"`    // 是否压缩旧日志文件
}

// LoadServerConfig 从指定路径加载服务器配置
func LoadServerConfig(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ServerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// GetPeers 线程安全地获取 Peers 列表
func (c *ServerConfig) GetPeers() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// 返回副本，避免外部修改影响内部数据
	peers := make([]string, len(c.Peers))
	copy(peers, c.Peers)
	return peers
}

// SetPeers 线程安全地设置 Peers 列表
func (c *ServerConfig) SetPeers(peers []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Peers = peers
}

// AddPeer 线程安全地添加一个 Peer
func (c *ServerConfig) AddPeer(peer string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if slices.Contains(c.Peers, peer) {
		return // 已存在，不重复添加
	}
	c.Peers = append(c.Peers, peer)
}

// Save 将当前配置保存到指定路径
func (c *ServerConfig) Save(path string) error {
	c.mu.RLock()
	data, err := json.MarshalIndent(c, "", "  ")
	c.mu.RUnlock()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
