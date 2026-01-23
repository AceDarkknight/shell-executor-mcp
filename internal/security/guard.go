package security

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"shell-executor-mcp/internal/logger"
)

// Guard 负责命令的安全审计
type Guard struct {
	blacklistedCommands []string
	dangerousArgsRegex  []*regexp.Regexp
}

// NewGuard 创建一个新的安全卫士实例
func NewGuard(blacklistedCommands []string, dangerousArgsRegex []string) (*Guard, error) {
	g := &Guard{
		blacklistedCommands: blacklistedCommands,
	}

	// 编译正则表达式
	for _, pattern := range dangerousArgsRegex {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern '%s': %v", pattern, err)
		}
		g.dangerousArgsRegex = append(g.dangerousArgsRegex, re)
	}

	return g, nil
}

// CheckCommand 检查命令是否安全
// 返回 error 表示命令被拦截，nil 表示命令安全
func (g *Guard) CheckCommand(cmd string) error {
	logger.Debugf("[DEBUG] Guard: 开始安全检查，命令: %s", cmd)

	if cmd == "" {
		logger.Debugf("[DEBUG] Guard: 命令为空")
		return errors.New("command is empty")
	}

	// 1. Trim & Normalize
	originalCmd := cmd
	cmd = strings.TrimSpace(cmd)
	cmd = strings.Join(strings.Fields(cmd), " ") // 压缩多余空格
	logger.Debugf("[DEBUG] Guard: 命令标准化，原始: %s, 标准化后: %s", originalCmd, cmd)

	// 2. Tokenize
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		logger.Debugf("[DEBUG] Guard: 标准化后命令为空")
		return errors.New("command is empty after normalization")
	}

	// 提取第一个 Token 作为 CommandVerb
	commandVerb := parts[0]
	logger.Debugf("[DEBUG] Guard: 提取命令动词: %s", commandVerb)

	// 3. Verb Check (黑名单检查)
	logger.Debugf("[DEBUG] Guard: 开始黑名单检查，黑名单: %v", g.blacklistedCommands)
	for _, blacklisted := range g.blacklistedCommands {
		if commandVerb == blacklisted {
			logger.Debugf("[DEBUG] Guard: 命令 '%s' 在黑名单中，拦截", commandVerb)
			return fmt.Errorf("command '%s' is blacklisted", commandVerb)
		}
	}
	logger.Debugf("[DEBUG] Guard: 黑名单检查通过")

	// 4. Args Check (危险参数检查)
	// 如果命令在黑名单中，已经在上面拦截了。
	// 这里检查那些虽然不在黑名单，但参数可能危险的命令。
	// 简单起见，我们检查整个命令字符串是否匹配危险正则。
	logger.Debugf("[DEBUG] Guard: 开始危险参数检查，正则数量: %d", len(g.dangerousArgsRegex))
	for _, re := range g.dangerousArgsRegex {
		if re.MatchString(cmd) {
			logger.Debugf("[DEBUG] Guard: 命令匹配危险正则: %s，拦截", re.String())
			return fmt.Errorf("command matches dangerous pattern: %s", re.String())
		}
	}
	logger.Debugf("[DEBUG] Guard: 危险参数检查通过")
	logger.Debugf("[DEBUG] Guard: 安全检查通过")

	return nil
}
