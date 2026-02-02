package configs

import (
	"encoding/json"
	"os"

	"github.com/AceDarkknight/shell-executor-mcp/internal/logger"
)

// ClientConfig 定义客户端的配置结构
type ClientConfig struct {
	Servers []ServerConfig `json:"servers"` // 服务器列表
	Token   string         `json:"token"`   // 连接 Token
	Log     LogConfig      `json:"log"`     // 日志配置
}

// ServerConfig 定义服务器的配置结构
type ServerConfig struct {
	Name string `json:"name"` // 服务器名称
	URL  string `json:"url"`  // 服务器地址
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

// ToLoggerConfig 转换为 logger.LogConfig
func (lc *LogConfig) ToLoggerConfig() logger.LogConfig {
	return logger.LogConfig{
		Level:      lc.Level,
		LogDir:     lc.LogDir,
		MaxSize:    lc.MaxSize,
		MaxBackups: lc.MaxBackups,
		MaxAge:     lc.MaxAge,
		Compress:   lc.Compress,
	}
}

// LoadClientConfig 从指定路径加载客户端配置
func LoadClientConfig(path string) (*ClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ClientConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
