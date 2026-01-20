# Logger 测试目录

## 目录说明

本目录包含日志功能的测试程序。

## 文件说明

- `main.go`: 日志功能测试程序，用于验证日志系统的初始化、日志输出和同步功能。

## 使用方法

编译并运行测试程序：

```bash
cd test/logger
go build -o test_logger.exe main.go
./test_logger.exe
```

## 测试内容

- 日志配置初始化
- 日志目录创建
- 日志文件初始化
- 不同级别日志输出（Info、Debug、Warn、Error）
- 日志同步

## 预期结果

程序将在日志目录下创建 `test.log` 文件，并输出测试日志信息。程序成功运行后会显示 "Logger test completed successfully!"。
