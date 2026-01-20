package executor

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"
)

// Executor 负责执行本地 Shell 命令
type Executor struct {
	// 可以在这里添加执行超时配置等
}

// NewExecutor 创建一个新的执行器实例
func NewExecutor() *Executor {
	return &Executor{}
}

// Result 表示命令执行的结果
type Result struct {
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
	Error    string `json:"error"`
}

// Execute 执行指定的 Shell 命令
// cmd: 要执行的命令字符串
// timeout: 执行超时时间，0 表示不限制
func (e *Executor) Execute(cmd string, timeout time.Duration) (*Result, error) {
	if cmd == "" {
		return nil, fmt.Errorf("command is empty")
	}

	// 使用 /bin/sh -c 来执行命令字符串
	// 注意：在 Windows 上可能需要调整为 cmd /c
	// 为了简化，这里假设是类 Unix 环境，或者后续添加 OS 检测
	var command *exec.Cmd
	if isWindows() {
		command = exec.Command("cmd", "/c", cmd)
	} else {
		command = exec.Command("/bin/sh", "-c", cmd)
	}

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	// 设置超时
	if timeout > 0 {
		timer := time.AfterFunc(timeout, func() {
			if command.Process != nil {
				command.Process.Kill()
			}
		})
		defer timer.Stop()
	}

	err := command.Run()

	result := &Result{
		Output: stdout.String(),
	}

	if err != nil {
		// 如果是超时导致的错误
		if timeout > 0 && err.Error() == "signal: killed" {
			result.Error = "execution timeout"
			result.ExitCode = -1
			return result, fmt.Errorf("command execution timeout")
		}

		// 获取退出码（如果有）
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		result.Error = err.Error()
	} else {
		result.ExitCode = 0
	}

	// 合并 stderr 到 error 字段，如果存在
	if stderr.Len() > 0 {
		if result.Error != "" {
			result.Error += "\n"
		}
		result.Error += stderr.String()
	}

	return result, nil
}

// isWindows 检测当前操作系统是否为 Windows
func isWindows() bool {
	// 简单的检测方法
	return len(exec.Command("cmd", "/c", "echo").String()) > 0
}
