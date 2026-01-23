package executor

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"shell-executor-mcp/internal/logger"
)

// Executor 负责执行本地 Shell 命令
type Executor struct {
	// 可以在这里添加执行超时配置等
	timeout time.Duration
}

// Result 表示命令执行的结果
type Result struct {
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
	Error    string `json:"error"`
}

// NewExecutor 创建一个新的执行器实例
func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) SetTimeout(timeout time.Duration) {
	e.timeout = timeout
}

// Execute 执行指定的 Shell 命令
// cmd: 要执行的命令字符串
// timeout: 执行超时时间，0 表示不限制
func (e *Executor) Execute(cmd string, timeout time.Duration) (*Result, error) {
	logger.Debugf("Executor: 开始执行命令: %s, 超时: %v\n", cmd, timeout)

	if cmd == "" {
		logger.Debugf("Executor: 命令为空")
		return nil, fmt.Errorf("command is empty")
	}

	// 创建输出缓冲区用于捕获标准输出和标准错误
	var stdout, stderr bytes.Buffer

	var command *exec.Cmd
	// 检测是否为 Windows 系统
	if isWindows() {
		logger.Debugf("Executor: 使用 Windows 命令: cmd /c\n")
		command = exec.Command("cmd", "/c", cmd)
	} else {
		logger.Debugf("Executor: 使用 Unix 命令: /bin/sh -c\n")
		command = exec.Command("/bin/sh", "-c", cmd)
	}

	// 设置命令的输出缓冲区
	command.Stdout = &stdout
	command.Stderr = &stderr

	// 设置超时
	var timer *time.Timer
	if timeout > 0 {
		logger.Debugf("Executor: 设置超时定时器: %v\n", timeout)
		timer = time.AfterFunc(timeout, func() {
			if command.Process != nil {
				logger.Debugf("Executor: 命令执行超时，终止进程\n")
				command.Process.Kill()
			}
		})
		// 确保在函数返回前停止 timer，避免资源泄漏
		defer timer.Stop()
	}

	logger.Debugf("Executor: 开始运行命令...\n")
	err := command.Run()

	result := &Result{
		Output: stdout.String(),
	}

	if err != nil {
		// 命令执行失败
		logger.Debugf("Executor: 命令执行失败, 错误: %v\n", err)
		result.Error = err.Error()
		result.ExitCode = -1

		// 如果是超时导致的错误
		if timeout > 0 && strings.Contains(err.Error(), "signal: killed") {
			logger.Debugf("Executor: 命令执行超时\n")
			result.Error = "execution timeout"
		}
	} else {
		// 命令执行成功，设置退出码为 0
		result.ExitCode = 0
	}

	logger.Debugf("Executor: 标准输出长度: %d\n", len(result.Output))

	// 合并 stderr 到 error 字段，如果存在
	if stderr.Len() > 0 {
		if result.Error != "" {
			result.Error += "\n"
		}
		result.Error += stderr.String()
		logger.Debugf("Executor: 合并标准错误到错误字段\n")
	}

	logger.Debugf("Executor: 命令执行完成, 退出码: %d\n", result.ExitCode)
	return result, nil
}

// isWindows 检测当前操作系统是否为 Windows
func isWindows() bool {
	return runtime.GOOS == "windows"
}
