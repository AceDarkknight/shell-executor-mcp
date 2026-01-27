package mcpclient

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Result 表示命令执行的结果
type Result struct {
	IsError bool                // 是否为错误结果
	Content []Content           // 内容列表
	Raw     *mcp.CallToolResult // 原始 MCP 结果
}

// Content 表示结果内容的接口
type Content interface {
	Type() string
}

// TextContent 表示文本内容
type TextContent struct {
	Text string
}

// Type 返回内容类型
func (tc *TextContent) Type() string {
	return "text"
}

// AggregatedResult 表示聚合结果（JSON 格式）
type AggregatedResult struct {
	Summary string            `json:"summary"` // 摘要
	Groups  []AggregatedGroup `json:"groups"`  // 组列表
}

// Type 返回内容类型
func (ar *AggregatedResult) Type() string {
	return "aggregated"
}

// AggregatedGroup 表示聚合结果中的一个组
type AggregatedGroup struct {
	Count  int      `json:"count"`  // 节点数量
	Status string   `json:"status"` // 状态
	Output string   `json:"output"` // 输出内容
	Error  string   `json:"error"`  // 错误信息
	Nodes  []string `json:"nodes"`  // 节点列表
}

// ParseResult 解析 MCP Tool 返回的结果
func ParseResult(result *mcp.CallToolResult) *Result {
	r := &Result{
		IsError: result.IsError,
		Raw:     result,
	}

	for _, content := range result.Content {
		switch v := content.(type) {
		case *mcp.TextContent:
			// 尝试解析为聚合结果
			var aggregatedResult AggregatedResult
			if err := json.Unmarshal([]byte(v.Text), &aggregatedResult); err == nil {
				// 成功解析为聚合结果
				r.Content = append(r.Content, &aggregatedResult)
			} else {
				// 作为纯文本内容
				r.Content = append(r.Content, &TextContent{Text: v.Text})
			}
		default:
			// 其他类型的内容，转换为文本
			// 使用 fmt.Sprintf 来转换
			r.Content = append(r.Content, &TextContent{Text: fmt.Sprintf("%v", v)})
		}
	}

	return r
}

// GetTextContents 获取所有文本内容
func (r *Result) GetTextContents() []string {
	var texts []string
	for _, content := range r.Content {
		if tc, ok := content.(*TextContent); ok {
			texts = append(texts, tc.Text)
		}
	}
	return texts
}

// GetAggregatedResults 获取所有聚合结果
func (r *Result) GetAggregatedResults() []*AggregatedResult {
	var results []*AggregatedResult
	for _, content := range r.Content {
		if ar, ok := content.(*AggregatedResult); ok {
			results = append(results, ar)
		}
	}
	return results
}

// String 返回结果的字符串表示
func (r *Result) String() string {
	var sb strings.Builder

	if r.IsError {
		sb.WriteString("Error Result:\n")
	} else {
		sb.WriteString("Success Result:\n")
	}

	for i, content := range r.Content {
		sb.WriteString(fmt.Sprintf("Content [%d]:\n", i))
		switch v := content.(type) {
		case *TextContent:
			sb.WriteString(v.Text)
			if !strings.HasSuffix(v.Text, "\n") {
				sb.WriteString("\n")
			}
		case *AggregatedResult:
			sb.WriteString(fmt.Sprintf("Summary: %s\n", v.Summary))
			for j, group := range v.Groups {
				sb.WriteString(fmt.Sprintf("  Group [%d]: count=%d, status=%s\n", j+1, group.Count, group.Status))
				if group.Output != "" {
					sb.WriteString(fmt.Sprintf("  Output:\n%s\n", group.Output))
				}
				if group.Error != "" {
					sb.WriteString(fmt.Sprintf("  Error: %s\n", group.Error))
				}
				if len(group.Nodes) > 0 {
					nodesStr := strings.Join(group.Nodes, ", ")
					if len(nodesStr) > 100 {
						nodesStr = nodesStr[:100] + "..."
					}
					sb.WriteString(fmt.Sprintf("  Nodes: %s\n", nodesStr))
				}
			}
		}
	}

	return sb.String()
}
