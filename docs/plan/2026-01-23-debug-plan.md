# 调试计划文档

## 1. 计划概述

本计划旨在在本地运行client和server进行debug，通过添加日志输出、编译、配置、启动和日志分析来验证系统逻辑的正确性。

## 2. 实现步骤

### 2.1 添加日志输出

#### 2.1.1 Client端日志增强
- **位置**: `cmd/client/cmd/run.go`
- **添加位置**:
  - `loadConfigFromViper()`: 添加配置加载日志，记录从viper读取的配置信息
  - `connectToAvailableServer()`: 添加连接过程的详细日志，包括尝试连接的服务器、连接结果、错误信息
  - `runCLI()`: 添加用户输入、命令发送、结果接收的详细日志
  - `displayResult()`: 添加结果解析和显示的日志

#### 2.1.2 Server端日志增强
- **位置**: `cmd/server/cmd/run.go`
- **添加位置**:
  - `loadConfigFromViper()`: 添加配置加载日志，记录从viper读取的配置信息
  - `runServer()`: 添加服务器启动各阶段的日志，包括配置加载、组件初始化、MCP Server创建、HTTP Handler注册、服务器启动
  - `internalExecHandler()`: 添加内部API请求处理的详细日志，包括请求接收、安全检查、命令执行、结果返回
  - `internalJoinHandler()`: 添加节点加入请求处理的日志
  - `internalSyncHandler()`: 添加节点同步请求处理的日志

#### 2.1.3 Dispatcher日志增强
- **位置**: `internal/dispatch/dispatcher.go`
- **添加位置**:
  - `Dispatch()`: 添加分发过程的日志，包括本地执行、远程分发、结果聚合
  - `executeOnPeer()`: 添加向peer节点发送请求的详细日志
  - `aggregateResults()`: 添加结果聚合的日志

#### 2.1.4 Executor日志增强
- **位置**: `internal/executor/executor.go`
- **添加位置**:
  - `Execute()`: 添加命令执行的详细日志，包括命令内容、执行结果、退出码、输出和错误信息

#### 2.1.5 Security日志增强
- **位置**: `internal/security/guard.go`
- **添加位置**:
  - `CheckCommand()`: 添加安全检查的详细日志，包括命令内容、检查结果、拦截原因

### 2.2 编译Client和Server

#### 2.2.1 编译命令
- **Client编译**: `go build -o bin/client.exe cmd/client/main.go`
- **Server编译**: `go build -o bin/server.exe cmd/server/main.go`

#### 2.2.2 预期效果
- 在`bin/`目录下生成`client.exe`和`server.exe`可执行文件
- 编译无错误

### 2.3 编写配置文件

#### 2.3.1 Client配置文件
- **文件路径**: `test/client_config.json`
- **配置内容**:
  - 服务器列表配置（指向本地server）
  - 日志配置（debug级别，日志目录为`logs/client`）

#### 2.3.2 Server配置文件
- **文件路径**: `test/server_config.json`
- **配置内容**:
  - 监听端口（8090）
  - 节点名称（node-01）
  - 集群token
  - 安全配置（黑名单命令和危险参数正则）
  - 日志配置（debug级别，日志目录为`logs/server`）

### 2.4 删除日志文件夹

#### 2.4.1 启动前删除
- **命令**: `Remove-Item -Recurse -Force logs` (Windows PowerShell)
- **目的**: 确保每次启动都是干净的日志环境，避免历史日志干扰

#### 2.4.2 启动后删除（下次启动前）
- **目的**: 保证多次启动互不影响

### 2.5 启动Server

#### 2.5.1 启动命令
- **命令**: `.\bin\server.exe --config test\server_config.json`
- **预期效果**:
  - 服务器成功启动
  - 监听在8090端口
  - 日志输出到`logs/server/server.log`
  - 显示启动信息和监听地址

#### 2.5.2 需要用户输入
- 无需用户输入，服务器启动后会持续运行

### 2.6 启动Client

#### 2.6.1 启动命令
- **命令**: `.\bin\client.exe --config test\client_config.json`
- **预期效果**:
  - 客户端成功启动
  - 连接到本地server（http://localhost:8090）
  - 日志输出到`logs/client/client.log`
  - 进入交互式命令行界面，显示提示符`>`

#### 2.6.2 需要用户输入
- 用户需要在命令行中输入要执行的shell命令
- 例如：`echo "hello world"`
- 输入`exit`或`quit`退出客户端

### 2.7 读取日志判断逻辑是否正确

#### 2.7.1 Server日志分析
- **检查点**:
  - 配置加载是否正确
  - MCP Server是否成功创建
  - HTTP Handler是否正确注册
  - 服务器是否成功启动并监听
  - 是否收到client的连接请求
  - 是否收到execute_command的调用
  - 安全检查是否正常执行
  - 命令执行是否成功
  - 结果是否正确返回

#### 2.7.2 Client日志分析
- **检查点**:
  - 配置加载是否正确
  - 连接到server是否成功
  - 命令发送是否成功
  - 结果接收是否成功
  - 结果解析和显示是否正确

#### 2.7.3 预期日志流程
1. Server启动，输出启动信息和监听地址
2. Client启动，输出配置信息和连接过程
3. Client成功连接到Server
4. 用户输入命令
5. Client发送命令到Server
6. Server接收命令，进行安全检查
7. Server执行命令
8. Server返回结果给Client
9. Client显示结果

## 3. 预期效果

### 3.1 成功场景
- Server和Client都成功启动
- Client成功连接到Server
- 用户输入的命令能够正确执行
- 结果能够正确显示
- 日志中记录了完整的执行流程
- 所有关键步骤都有对应的日志输出

### 3.2 可能的问题场景
- **连接失败**: 检查端口是否正确、server是否启动
- **命令执行失败**: 检查命令是否合法、安全检查是否误拦截
- **结果解析失败**: 检查返回结果格式是否正确
- **日志输出缺失**: 检查日志配置是否正确

## 4. 注意事项

1. **日志级别**: 使用debug级别确保所有日志都能输出
2. **日志目录**: 确保日志目录存在且有写权限
3. **端口占用**: 确保8090端口没有被其他程序占用
4. **命令安全性**: 测试时避免使用危险命令
5. **Windows兼容性**: 注意Windows和Unix系统的命令差异
6. **并发安全**: 日志输出本身是并发安全的，但要注意日志内容的准确性

## 5. 测试用例

### 5.1 基础测试
- 测试命令: `echo "hello world"`
- 预期输出: `hello world`

### 5.2 多行输出测试
- 测试命令: `dir` (Windows) 或 `ls -la` (Unix)
- 预期输出: 目录列表

### 5.3 错误命令测试
- 测试命令: `invalid_command`
- 预期输出: 错误信息

### 5.4 安全检查测试
- 测试命令: `rm -rf /` (应该被拦截)
- 预期输出: 安全拦截错误信息
