package configs

import (
	"encoding/json"
	"os"

	"shell-executor-mcp/internal/logger"
)

// ServerConfig 定义单个服务器配置
type ServerConfig struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// ClientConfig 定义客户端配置
type ClientConfig struct {
	Servers []ServerConfig `json:"servers"`
	Log     LogConfig      `json:"log"` // 日志配置
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

// ToLoggerConfig 将配置转换为 logger.LogConfig
func (c *LogConfig) ToLoggerConfig() *logger.LogConfig {
	if c == nil {
		return logger.DefaultLogConfig()
	}
	return &logger.LogConfig{
		Level:      c.Level,
		LogDir:     c.LogDir,
		MaxSize:    c.MaxSize,
		MaxBackups: c.MaxBackups,
		MaxAge:     c.MaxAge,
		Compress:   c.Compress,
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
