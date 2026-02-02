package mcpclient

import (
	"github.com/AceDarkknight/shell-executor-mcp/internal/logger"
)

// Logger 定义日志记录器接口
type Logger interface {
	Debugf(template string, args ...interface{})
	Infof(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
}

// defaultLogger 是默认的日志记录器实现，使用 logger 包
type defaultLogger struct{}

// Debugf 记录 Debug 级别日志
func (l *defaultLogger) Debugf(template string, args ...interface{}) {
	logger.Debugf(template, args...)
}

// Infof 记录 Info 级别日志
func (l *defaultLogger) Infof(template string, args ...interface{}) {
	logger.Infof(template, args...)
}

// Warnf 记录 Warn 级别日志
func (l *defaultLogger) Warnf(template string, args ...interface{}) {
	logger.Warnf(template, args...)
}

// Errorf 记录 Error 级别日志
func (l *defaultLogger) Errorf(template string, args ...interface{}) {
	logger.Errorf(template, args...)
}

// GetDefaultLogger 返回默认的日志记录器
func GetDefaultLogger() Logger {
	return &defaultLogger{}
}
