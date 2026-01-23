# 命令执行器模块 (executor)

## 概述

命令执行器模块负责在本地 Shell 环境中执行命令，并捕获执行结果（包括退出码、标准输出和标准错误）。该模块支持超时控制，防止长时间运行的命令阻塞系统。

## 文件说明

- `executor.go` - 执行器实现，包含命令执行逻辑

## 数据结构

### Executor

执行器结构，用于执行本地 Shell 命令。

### Result

命令执行结果，包含以下字段：

- `ExitCode` - 命令退出码（0 表示成功）
- `Output` - 标准输出
- `Error` - 错误信息（包括 stderr 和执行错误）

## 主要功能

1. **命令执行**
   - 支持 Unix 和 Windows 系统
   - Unix: 使用 `/bin/sh -c` 执行命令
   - Windows: 使用 `cmd /c` 执行命令

2. **超时控制**
   - 支持设置执行超时时间
   - 超时后自动终止命令进程
   - 超时时返回错误信息

3. **结果捕获**
   - 捕获标准输出（stdout）
   - 捕获标准错误（stderr）
   - 获取命令退出码

4. **错误处理**
   - 处理命令执行失败的情况
   - 区分超时错误和其他错误
   - 合并 stderr 到错误信息中

## 使用示例

```go
// 创建执行器
executor := executor.NewExecutor()

// 执行命令（带超时）
result, err := executor.Execute("echo Hello World", 5*time.Second)
if err != nil {
    log.Printf("Execution failed: %v", err)
}

// 查看结果
fmt.Printf("Exit Code: %d\n", result.ExitCode)
fmt.Printf("Output: %s\n", result.Output)
if result.Error != "" {
    fmt.Printf("Error: %s\n", result.Error)
}

// 执行命令（无超时）
result, err := executor.Execute("ls -la", 0)
```

## 超时处理

当设置超时时间时：

1. 使用 `time.AfterFunc` 创建定时器
2. 超时后调用 `Process.Kill()` 终止进程
3. 返回超时错误信息

```go
// 5秒超时
result, err := executor.Execute("sleep 10", 5*time.Second)
// err 会包含 "command execution timeout" 错误
```

## 跨平台支持

执行器会自动检测操作系统类型：

- **Unix/Linux/macOS**: 使用 `/bin/sh -c`
- **Windows**: 使用 `cmd /c`

## 安全注意事项

⚠️ **重要**: 执行器本身不进行安全检查，安全检查应由调用方（如 security.Guard）在调用执行器之前完成。

## 更新记录

- 2026-01-23: 创建 README.md 文档
