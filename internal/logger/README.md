# Logger 模块

## 概述

Logger 模块提供基于 zap 的日志记录功能，支持日志轮转、多级别日志记录，并保证并发安全。

## 功能特性

- 基于 Uber Zap 的高性能日志记录
- 支持日志轮转（使用 lumberjack）
- 支持多种日志级别：debug、info、warn、error、fatal
- 并发安全，支持多 goroutine 同时调用
- 支持 JSON 格式日志输出
- 支持结构化日志字段
- 自动添加调用者信息和时间戳

## 并发安全设计

### 实现原理

Logger 模块使用 `sync.Once` 保证并发安全：

1. **InitLogger 函数**
   - 使用 `sync.Once.Do()` 包装初始化逻辑
   - 确保即使多个 goroutine 同时调用，初始化代码也只会执行一次
   - 避免了全局变量赋值的竞态条件

2. **L() 和 S() 函数**
   - 使用 `sync.Once.Do()` 实现懒加载
   - 如果 logger 未初始化，会自动使用默认配置初始化
   - 保证并发安全的同时，提供便捷的使用方式

### 既是 client 又是 server 的场景

当一个节点同时作为 client 和 server 运行时，推荐使用以下方式：

#### 方案：使用同一个 logger 文件，通过字段区分

```go
// 初始化 logger（只需初始化一次）
cfg := &logger.LogConfig{
    Level:      "debug",
    LogDir:     "logs",
    MaxSize:    100,
    MaxBackups: 3,
    MaxAge:     28,
    Compress:   false,
}
err := logger.InitLogger(cfg, "app.log")
if err != nil {
    log.Fatal(err)
}

// Client 日志，添加 role 字段区分
logger.Info("Client started", zap.String("role", "client"))
logger.Info("Sending request", zap.String("role", "client"), zap.String("url", "http://example.com"))

// Server 日志，添加 role 字段区分
logger.Info("Server started", zap.String("role", "server"))
logger.Info("Received request", zap.String("role", "server"), zap.String("path", "/api"))
```

**优点：**
- 所有日志集中在同一个文件，便于查看和分析
- 通过 `role` 字段可以轻松过滤和查询特定角色的日志
- 符合并发安全设计，不会出现日志混乱或丢失

## 使用方法

### 1. 初始化 Logger

```go
import "your-project/internal/logger"

// 使用默认配置初始化
err := logger.InitLogger(nil, "app.log")
if err != nil {
    log.Fatal(err)
}

// 使用自定义配置初始化
cfg := &logger.LogConfig{
    Level:      "debug",  // 日志级别
    LogDir:     "logs",   // 日志目录
    MaxSize:    100,      // 单个文件最大大小（MB）
    MaxBackups: 3,        // 保留的旧文件数量
    MaxAge:     28,       // 保留旧文件的最大天数
    Compress:   false,    // 是否压缩旧文件
}
err := logger.InitLogger(cfg, "app.log")
if err != nil {
    log.Fatal(err)
}
```

### 2. 记录日志

#### 使用结构化日志（推荐）

```go
import "go.uber.org/zap"

// Info 级别
logger.Info("User logged in",
    zap.String("user_id", "12345"),
    zap.String("ip", "192.168.1.1"),
)

// Debug 级别
logger.Debug("Processing request",
    zap.String("method", "GET"),
    zap.String("path", "/api/users"),
)

// Warn 级别
logger.Warn("Slow query detected",
    zap.Duration("duration", time.Second*5),
    zap.String("query", "SELECT * FROM users"),
)

// Error 级别
logger.Error("Failed to connect to database",
    zap.Error(err),
    zap.String("host", "localhost"),
)

// Fatal 级别（会退出程序）
logger.Fatal("Critical error",
    zap.Error(err),
)
```

#### 使用格式化日志

```go
// Infof
logger.Infof("User %s logged in from %s", "john", "192.168.1.1")

// Debugf
logger.Debugf("Processing %s request to %s", "GET", "/api/users")

// Warnf
logger.Warnf("Query took %v to complete", time.Second*5)

// Errorf
logger.Errorf("Failed to connect: %v", err)

// Fatalf
logger.Fatalf("Critical error: %v", err)
```

### 3. 同步日志缓冲区

在程序退出前，建议调用 `Sync()` 确保所有日志都写入文件：

```go
defer logger.Sync()
```

## 配置说明

### LogConfig 结构体

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| Level | string | "debug" | 日志级别：debug、info、warn、error |
| LogDir | string | "logs" | 日志文件目录 |
| MaxSize | int | 100 | 单个日志文件最大大小（MB） |
| MaxBackups | int | 3 | 保留的旧日志文件最大数量 |
| MaxAge | int | 28 | 保留旧日志文件的最大天数 |
| Compress | bool | false | 是否压缩旧日志文件 |

## 测试

运行单元测试：

```bash
cd internal/logger
go test -v
```

运行并发测试：

```bash
go test -v -run TestConcurrent
```

## 注意事项

1. **并发安全**
   - Logger 模块已实现并发安全，可以在多个 goroutine 中安全使用
   - `InitLogger` 使用 `sync.Once` 保证只初始化一次
   - 多次调用 `InitLogger` 不会产生错误，但只有第一次调用生效

2. **日志文件路径**
   - 日志文件路径为 `{LogDir}/{filename}`
   - 确保程序有权限在 `LogDir` 目录下创建和写入文件

3. **性能考虑**
   - Zap 是高性能日志库，适合高并发场景
   - 建议在生产环境中使用 `Info` 级别，避免过多的 `Debug` 日志

4. **日志轮转**
   - 使用 lumberjack 自动进行日志轮转
   - 当日志文件达到 `MaxSize` 时会自动创建新文件
   - 旧文件会根据 `MaxBackups` 和 `MaxAge` 自动清理

5. **既是 client 又是 server 的场景**
   - 推荐使用同一个 logger 文件，通过 `role` 字段区分
   - 不要尝试创建多个 logger 实例，因为 `sync.Once` 不支持重置
   - 如果确实需要多个独立的 logger，需要修改架构设计

## 示例

完整的示例代码请参考 `test/logger/main.go`。

## 依赖

- go.uber.org/zap
- go.uber.org/zap/zapcore
- gopkg.in/natefinch/lumberjack.v2
