package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogConfig 日志配置
type LogConfig struct {
	Level      string // 日志级别: debug, info, warn, error
	LogDir     string // 日志文件目录，默认为当前目录
	MaxSize    int    // 单个日志文件最大大小（MB），默认 100MB
	MaxBackups int    // 保留的旧日志文件最大数量，默认 3 个
	MaxAge     int    // 保留旧日志文件的最大天数，默认 28 天
	Compress   bool   // 是否压缩旧日志文件，默认 false
}

// DefaultLogConfig 返回默认的日志配置
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:      "debug",
		LogDir:     "logs",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   false,
	}
}

var (
	globalLogger *zap.Logger
	sugarLogger  *zap.SugaredLogger
)

// InitLogger 初始化全局日志记录器
// cfg: 日志配置
// filename: 日志文件名（如 "server.log" 或 "client.log"）
func InitLogger(cfg *LogConfig, filename string) error {
	if cfg == nil {
		cfg = DefaultLogConfig()
	}

	// 确保日志目录存在
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// 构建日志文件完整路径
	logFilePath := filepath.Join(cfg.LogDir, filename)

	// 配置 lumberjack 进行日志轮转
	fileWriter := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}

	// 解析日志级别
	level := parseLogLevel(cfg.Level)

	// 配置编码器（包含时间戳、日志级别、调用信息等）
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建核心
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(fileWriter),
		level,
	)

	// 创建全局日志记录器
	globalLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))
	sugarLogger = globalLogger.Sugar()

	return nil
}

// parseLogLevel 解析日志级别字符串
func parseLogLevel(levelStr string) zapcore.Level {
	switch levelStr {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// L 返回全局的 zap.Logger
func L() *zap.Logger {
	if globalLogger == nil {
		// 如果未初始化，使用默认配置初始化
		_ = InitLogger(nil, "default.log")
	}
	return globalLogger
}

// S 返回全局的 zap.SugaredLogger
func S() *zap.SugaredLogger {
	if sugarLogger == nil {
		// 如果未初始化，使用默认配置初始化
		_ = InitLogger(nil, "default.log")
	}
	return sugarLogger
}

// Sync 同步日志缓冲区
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// Debug 记录 Debug 级别日志
func Debug(msg string, fields ...zap.Field) {
	L().Debug(msg, fields...)
}

// Debugf 记录 Debug 级别日志（格式化）
func Debugf(template string, args ...interface{}) {
	S().Debugf(template, args...)
}

// Info 记录 Info 级别日志
func Info(msg string, fields ...zap.Field) {
	L().Info(msg, fields...)
}

// Infof 记录 Info 级别日志（格式化）
func Infof(template string, args ...interface{}) {
	S().Infof(template, args...)
}

// Warn 记录 Warn 级别日志
func Warn(msg string, fields ...zap.Field) {
	L().Warn(msg, fields...)
}

// Warnf 记录 Warn 级别日志（格式化）
func Warnf(template string, args ...interface{}) {
	S().Warnf(template, args...)
}

// Error 记录 Error 级别日志
func Error(msg string, fields ...zap.Field) {
	L().Error(msg, fields...)
}

// Errorf 记录 Error 级别日志（格式化）
func Errorf(template string, args ...interface{}) {
	S().Errorf(template, args...)
}

// Fatal 记录 Fatal 级别日志并退出程序
func Fatal(msg string, fields ...zap.Field) {
	L().Fatal(msg, fields...)
}

// Fatalf 记录 Fatal 级别日志（格式化）并退出程序
func Fatalf(template string, args ...interface{}) {
	S().Fatalf(template, args...)
}
