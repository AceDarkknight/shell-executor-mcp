# 开发计划：日志系统集成 (2026-01-20)

## 目标
为 Shell Executor MCP 系统的 Client 和 Server 端集成结构化日志功能，支持日志级别配置、文件分割与自动归档。

## 需求分析
1. **日志输出**：
   - Client 输出到 `client.log`。
   - Server 输出到 `server.log`。
   - 支持配置日志文件夹，默认为当前运行路径。
2. **日志级别**：
   - 支持 `debug`, `info`, `warn`, `error`。
   - 可通过配置或启动参数控制。
3. **日志格式**：
   - 必须包含：时间戳、日志级别、日志内容、调用函数名、文件名、行号。
4. **日志轮转**：
   - 按日期或大小自动分割和归档。
5. **规范遵循**：
   - 遵循 `编程流程规范.md`，使用 `zap` 作为日志库，`lumberjack` 进行日志轮转。

## 步骤

### 第一阶段：依赖与配置
- [ ] 1. 添加依赖：
  - `go.uber.org/zap` (结构化日志)
  - `gopkg.in/natefinch/lumberjack.v2` (日志轮转)
- [ ] 2. 更新配置结构体 (`internal/config` 和 `pkg/configs`)：
  - 增加 `LogConfig` 结构，包含 `Level`, `LogDir` 等字段。

### 第二阶段：封装 Logger 模块
- [ ] 3. 创建 `internal/logger` 包：
  - 实现 `InitLogger(cfg LogConfig, filename string) error` 函数。
  - 配置 `zapcore` 以满足格式要求（时间、调用方、级别等）。
  - 配置 `lumberjack.Logger` 作为 `WriteSyncer` 实现文件轮转。
  - 暴露全局 Logger 或提供获取 Logger 的单例方法。

### 第三阶段：集成到应用
- [ ] 4. 修改 `cmd/server/main.go`：
  - 在启动时加载配置后初始化 Logger。
  - 替换原有的 `log.Printf` 为 `zap.L().Info/Error` 等。
  - 添加关键流程的日志埋点（启动、请求接收、分发、结果聚合）。
- [ ] 5. 修改 `cmd/client/main.go`：
  - 在启动时初始化 Logger。
  - 替换 `log.Printf`。
  - 记录用户输入、连接状态、错误信息。

### 第四阶段：测试与验证
- [ ] 6. 编写单元测试验证 Logger 配置是否生效。
- [ ] 7. 运行 `test_single_node.sh` 并检查生成的日志文件内容格式。
- [ ] 8. 验证日志轮转功能（通过设置较小的轮转阈值进行测试）。

## 预期效果
- 运行程序后，指定目录下生成 `server.log` 和 `client.log`。
- 日志内容示例：
  `2026-01-20T10:00:00.000+0800 INFO internal/dispatch/dispatcher.go:50 Dispatching command {"command": "echo hello", "caller": "dispatch.Dispatch"}`
- 日志文件按配置进行轮转。
