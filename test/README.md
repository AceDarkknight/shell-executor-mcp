# 测试目录

## 目录说明

本目录包含项目的测试程序和配置文件。

## 目录结构

- `logger/`: 日志功能测试目录
  - `main.go`: 日志功能测试程序
  - `README.md`: 日志测试说明文档
- `simple_server_test.go`: 简单的服务器测试程序（不使用日志）
- `client_config.json`: 客户端配置文件
- `server_config.json`: 服务器配置文件
- `test_server_config.json`: 测试用服务器配置文件

## 测试程序

### 日志测试

进入 `logger` 目录，编译并运行日志测试程序：

```bash
cd test/logger
go build -o test_logger.exe main.go
./test_logger.exe
```

### 简单服务器测试

编译并运行简单服务器测试程序：

```bash
cd test
go build -o simple_server_test.exe simple_server_test.go
./simple_server_test.exe
```

## 配置文件

- `client_config.json`: 用于配置客户端连接的参数
- `server_config.json`: 用于配置服务器运行的参数
- `test_server_config.json`: 测试环境的服务器配置

## 注意事项

- 测试程序主要用于验证各个模块的功能
- 运行测试前请确保已正确配置相关配置文件
- 测试产生的日志文件默认存放在 `logs` 目录下
