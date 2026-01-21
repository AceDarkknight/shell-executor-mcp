# Logger 并发安全修复计划

## 日期
2026-01-21

## 问题描述

### 当前问题分析

在 `internal/logger/logger.go` 中存在以下并发安全问题：

1. **InitLogger 函数的竞态条件**
   - 第92-93行直接赋值全局变量 `globalLogger` 和 `sugarLogger`
   - 没有任何同步机制保护这些全局变量的赋值操作
   - 多个 goroutine 同时调用 `InitLogger` 会导致竞态条件

2. **节点既是 client 又是 server 的问题**
   - 如果同一个节点同时作为 client 和 server 运行
   - 会分别调用 `InitLogger` 并传入不同的 `filename`（如 "server.log" 和 "client.log"）
   - 后调用的会覆盖先调用的 logger 实例
   - 导致日志混乱、丢失或写入错误的文件

3. **L() 和 S() 函数的潜在问题**
   - 第116-118行和第125-127行在检查到 nil 时会自动调用 InitLogger
   - 这可能在多个 goroutine 同时访问时触发竞态条件

## 影响范围

- 所有使用 logger 的模块都可能受影响
- 在集群部署或单节点同时运行 client 和 server 时问题尤为明显
- 可能导致日志丢失、日志文件混乱、程序崩溃等严重问题

## 解决方案

### 方案一：使用 sync.Once 保护初始化（推荐）

**优点：**
- 代码改动最小
- sync.Once 保证 InitLogger 只执行一次
- 简单有效，符合 Go 最佳实践

**缺点：**
- 如果需要不同的日志文件（如 client.log 和 server.log），此方案不适用
- 需要明确是使用单例 logger 还是支持多个 logger 实例

**实现步骤：**
1. 添加 `sync.Once` 变量保护初始化
2. 在 InitLogger 中使用 once.Do 确保只执行一次
3. 在 L() 和 S() 中也使用 once.Do 确保懒加载安全

### 方案二：使用 sync.Mutex 保护全局变量（次选）

**优点：**
- 允许重新初始化 logger（如果需要）
- 更灵活的控制

**缺点：**
- 需要在每次访问时加锁，性能略低
- 代码改动相对较大

### 方案三：支持多个 logger 实例（如果需要）

**优点：**
- 可以同时支持 client 和 server 的独立日志
- 更灵活的日志管理

**缺点：**
- 代码改动最大
- 需要修改所有调用 logger 的地方
- 需要明确如何区分不同的 logger 实例

## 推荐方案

**采用方案一（sync.Once）**，理由如下：

1. 符合"代码改动最小"的原则
2. 对于大多数场景，单例 logger 已经足够
3. 如果确实需要区分 client 和 server 的日志，可以通过在日志中添加字段来区分

## 实现步骤

### 1. 修改 logger.go

#### 添加 sync.Once 变量
```go
var (
    globalLogger *zap.Logger
    sugarLogger  *zap.SugaredLogger
    loggerOnce   sync.Once
)
```

#### 修改 InitLogger 函数
- 将初始化逻辑封装到闭包中
- 使用 once.Do 确保只执行一次
- 如果已经初始化过，可以选择返回错误或忽略

#### 修改 L() 和 S() 函数
- 使用 once.Do 确保懒加载安全

### 2. 对于既是 client 又是 server 的场景

**方案 A：使用同一个 logger 文件**
- 所有日志写入同一个文件（如 "app.log"）
- 通过在日志中添加字段区分 client 和 server
- 例如：`logger.Info("message", zap.String("role", "client"))`

**方案 B：使用不同的 logger 实例**（如果确实需要）
- 修改架构，支持多个 logger 实例
- client 和 server 分别使用各自的 logger
- 需要更大的代码改动

### 3. 编写单元测试

#### 测试用例1：并发调用 InitLogger
- 启动多个 goroutine 同时调用 InitLogger
- 验证只有一个 goroutine 成功初始化
- 验证没有竞态条件

#### 测试用例2：并发调用 L() 和 S()
- 启动多个 goroutine 同时调用 L() 和 S()
- 验证懒加载的安全性

#### 测试用例3：既是 client 又是 server 的场景
- 模拟同时初始化 client 和 server logger
- 验证日志写入正确

### 4. 更新文档

- 更新 `internal/logger/README.md`，说明并发安全的实现
- 更新 `docs/architecture.md`，说明日志模块的设计
- 如果采用方案 A，说明如何区分 client 和 server 的日志

## 预期效果

1. **并发安全**
   - 多个 goroutine 同时调用 InitLogger 不会产生竞态条件
   - L() 和 S() 函数的懒加载也是并发安全的

2. **单节点既是 client 又是 server**
   - 如果采用方案 A，所有日志写入同一个文件，通过字段区分
   - 不会出现日志丢失或混乱的情况

3. **代码质量**
   - 符合 Go 并发编程最佳实践
   - 代码改动最小，风险可控

## 风险评估

- **低风险**：使用 sync.Once 是 Go 标准库推荐的并发安全模式
- **兼容性**：对外接口保持不变，不影响现有代码
- **性能**：sync.Once 的性能开销极小，几乎可以忽略

## 需要用户确认的问题

1. **是否需要支持多个独立的 logger 实例？**
   - 如果需要，client 和 server 可以使用不同的日志文件
   - 如果不需要，使用同一个 logger 文件，通过字段区分

2. **对于既是 client 又是 server 的场景，更倾向于哪种方案？**
   - 方案 A：同一个 logger 文件，通过字段区分
   - 方案 B：不同的 logger 实例，不同的日志文件

3. **是否需要在 InitLogger 被重复调用时返回错误？**
   - 返回错误可以让调用者知道已经初始化过
   - 忽略可以让调用者无感知
