# 模块路径重命名计划

**日期**: 2026-02-02
**状态**: 待确认

## 1. 背景与目标

当前项目的 module path 为 `shell-executor-mcp`。为了符合 Go 社区标准以及支持 `go get` 等远程获取方式，需要将其更改为 `github.com/AceDarkknight/shell-executor-mcp`。

## 2. 变更信息

- **当前 Module Path**: `shell-executor-mcp`
- **目标 Module Path**: `github.com/AceDarkknight/shell-executor-mcp`

## 3. 执行步骤

### 3.1 修改 go.mod

将 `go.mod` 文件第一行：
```go
module shell-executor-mcp
```
修改为：
```go
module github.com/AceDarkknight/shell-executor-mcp
```

### 3.2 全局替换 Import 路径

在项目根目录下，对所有 `.go` 文件执行全局替换操作。

- **查找内容**: `"shell-executor-mcp/`
- **替换为**: `"github.com/AceDarkknight/shell-executor-mcp/`

**注意**: 需要处理好边界情况，确保只替换作为包导入前缀的部分。由于原 module 名较简单，建议使用明确的字符串匹配（带双引号）以避免误伤。

涉及的主要目录：
- `cmd/`
- `pkg/`
- `internal/`
- `test/`

### 3.3 清理与更新依赖

在替换完成后，执行：
```bash
go mod tidy
```
确保 `go.sum` 更新，且没有遗漏的依赖。

### 3.4 编译验证

验证 Server 和 Client 是否能正常编译：

```bash
# 编译 Server
go build -o bin/server.exe ./cmd/server
# 验证 Server 版本/帮助信息 (如果支持)
./bin/server.exe --help

# 编译 Client
go build -o bin/client.exe ./cmd/client
# 验证 Client 帮助信息
./bin/client.exe --help
```

### 3.5 功能验证

1.  **单元测试**:
    运行所有单元测试确保逻辑未破坏。
    ```bash
    go test ./...
    ```

2.  **集成验证**:
    - 启动 Server：
      ```bash
      ./bin/server.exe
      ```
    - 使用 Client 连接测试（包括 Debug 模式）：
      ```bash
      # 普通运行
      ./bin/client.exe run "echo hello"
      
      # Debug 连接 (确保之前的 debug 功能正常)
      ./bin/client.exe run "echo hello" --debug
      ```
    - 检查日志输出，确认没有奇怪的路径报错。

## 4. 回滚计划

如果变更导致无法解决的编译错误或运行时严重问题：

1.  **还原 go.mod**: 将 module path 改回 `shell-executor-mcp`。
2.  **反向替换**: 将所有 `.go` 文件中的 `"github.com/AceDarkknight/shell-executor-mcp/` 替换回 `"shell-executor-mcp/`。
3.  **运行 go mod tidy**: 恢复依赖状态。
