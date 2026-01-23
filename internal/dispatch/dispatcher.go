package dispatch

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"shell-executor-mcp/internal/executor"
	"shell-executor-mcp/internal/logger"
)

// Dispatcher 负责将命令分发给集群节点并聚合结果
type Dispatcher struct {
	peers      []string
	token      string
	httpClient *http.Client
}

// NewDispatcher 创建一个新的分发器实例
func NewDispatcher(peers []string, token string) *Dispatcher {
	return &Dispatcher{
		peers: peers,
		token: token,
		httpClient: &http.Client{
			Timeout: 5 * time.Second, // 默认5秒超时
		},
	}
}

// NodeResult 表示单个节点的执行结果
type NodeResult struct {
	NodeName string `json:"node_name"`
	Status   string `json:"status"` // success, failed, timeout
	Output   string `json:"output"`
	Error    string `json:"error"`
}

// AggregatedGroup 聚合后的结果组
type AggregatedGroup struct {
	Output string   `json:"output"`
	Error  string   `json:"error"`
	Status string   `json:"status"`
	Nodes  []string `json:"nodes"`
	Count  int      `json:"count"`
}

// DispatchRequest 分发请求的 Body 结构
type DispatchRequest struct {
	Cmd string `json:"cmd"`
}

// DispatchResponse 分发响应的 Body 结构
type DispatchResponse struct {
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
	Error    string `json:"error"`
}

// Dispatch 执行命令分发和聚合
// localExecutor: 本地执行器
// nodeName: 当前节点名称
// cmd: 要执行的命令
func (d *Dispatcher) Dispatch(localExecutor *executor.Executor, nodeName string, cmd string) ([]AggregatedGroup, string) {
	// 导入 logger
	// 这里需要导入 logger 包，但由于代码结构限制，暂时使用 fmt 输出
	// 在实际使用中，应该在文件顶部导入 logger 包

	logger.Infof("Dispatcher: 开始分发命令: %s, 节点名称: %s\n", cmd, nodeName)
	logger.Infof("Dispatcher: Peer 节点数量: %d\n", len(d.peers))

	// 用于收集所有结果
	var results []NodeResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 1. 本地执行
	logger.Infof("Dispatcher: 开始本地执行\n")
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Infof("Dispatcher: 执行命令: %s, 超时: 5s\n", cmd)
		res, err := localExecutor.Execute(cmd, 5*time.Second)
		if err != nil {
			logger.Infof("Dispatcher: 本地执行失败: %v\n", err)
			mu.Lock()
			results = append(results, NodeResult{
				NodeName: nodeName,
				Status:   "failed",
				Error:    err.Error(),
			})
			mu.Unlock()
			return
		}

		logger.Infof("Dispatcher: 本地执行成功, 退出码: %d, 输出长度: %d\n", res.ExitCode, len(res.Output))
		mu.Lock()
		results = append(results, NodeResult{
			NodeName: nodeName,
			Status:   "success",
			Output:   res.Output,
			Error:    res.Error,
		})
		mu.Unlock()
	}()

	// 2. 分发给其他节点
	logger.Infof("Dispatcher: 开始向 %d 个 peer 节点分发\n", len(d.peers))
	for i, peer := range d.peers {
		wg.Add(1)
		go func(peerURL string, index int) {
			defer wg.Done()
			logger.Infof("Dispatcher: 向 peer [%d] 发送请求: %s\n", index+1, peerURL)
			result := d.executeOnPeer(peerURL, cmd)
			logger.Infof("Dispatcher: peer [%d] 执行完成, 状态: %s\n", index+1, result.Status)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(peer, i)
	}

	// 等待所有任务完成
	logger.Infof("Dispatcher: 等待所有任务完成...\n")
	wg.Wait()
	logger.Infof("Dispatcher: 所有任务完成, 结果数量: %d\n", len(results))

	// 3. 聚合结果
	logger.Infof("Dispatcher: 开始聚合结果\n")
	groups := d.aggregateResults(results)
	summary := fmt.Sprintf("Executed on %d nodes, %d groups found", len(results), len(groups))
	logger.Infof("Dispatcher: 聚合完成, 组数: %d, 摘要: %s\n", len(groups), summary)

	return groups, summary
}

// executeOnPeer 在指定的 Peer 节点上执行命令
func (d *Dispatcher) executeOnPeer(peerURL string, cmd string) NodeResult {
	logger.Infof("executeOnPeer: 开始向 peer 执行命令, peerURL: %s, cmd: %s\n", peerURL, cmd)

	reqBody := DispatchRequest{Cmd: cmd}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		logger.Infof("executeOnPeer: 序列化请求失败: %v\n", err)
		return NodeResult{
			NodeName: peerURL,
			Status:   "failed",
			Error:    fmt.Sprintf("marshal request failed: %v", err),
		}
	}
	logger.Infof("executeOnPeer: 请求体序列化成功\n")

	// 构建请求 URL
	// peerURL 应该是完整的 http://host:port
	url := fmt.Sprintf("%s/internal/exec", peerURL)
	logger.Infof("executeOnPeer: 构建请求 URL: %s\n", url)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Infof("executeOnPeer: 创建请求失败: %v\n", err)
		return NodeResult{
			NodeName: peerURL,
			Status:   "failed",
			Error:    fmt.Sprintf("create request failed: %v", err),
		}
	}

	req.Header.Set("Content-Type", "application/json")
	if d.token != "" {
		req.Header.Set("X-Cluster-Token", d.token)
		logger.Infof("executeOnPeer: 设置 Cluster Token\n")
	}

	logger.Infof("executeOnPeer: 发送 HTTP 请求...\n")
	resp, err := d.httpClient.Do(req)
	if err != nil {
		logger.Infof("executeOnPeer: HTTP 请求失败: %v\n", err)
		return NodeResult{
			NodeName: peerURL,
			Status:   "failed",
			Error:    fmt.Sprintf("request failed: %v", err),
		}
	}
	defer resp.Body.Close()
	logger.Infof("executeOnPeer: HTTP 请求成功, 状态码: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Infof("executeOnPeer: 服务器返回错误状态码, body: %s\n", string(body))
		return NodeResult{
			NodeName: peerURL,
			Status:   "failed",
			Error:    fmt.Sprintf("server returned %d: %s", resp.StatusCode, string(body)),
		}
	}

	var respData DispatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		logger.Infof("executeOnPeer: 解析响应失败: %v\n", err)
		return NodeResult{
			NodeName: peerURL,
			Status:   "failed",
			Error:    fmt.Sprintf("decode response failed: %v", err),
		}
	}
	logger.Infof("executeOnPeer: 响应解析成功, 退出码: %d, 输出长度: %d\n", respData.ExitCode, len(respData.Output))

	status := "success"
	if respData.ExitCode != 0 || respData.Error != "" {
		status = "failed"
		logger.Infof("executeOnPeer: 命令执行失败, 退出码: %d, 错误: %s\n", respData.ExitCode, respData.Error)
	}

	logger.Infof("executeOnPeer: peer 执行完成\n")
	return NodeResult{
		NodeName: peerURL,
		Status:   status,
		Output:   respData.Output,
		Error:    respData.Error,
	}
}

// aggregateResults 将结果按输出内容进行分组压缩
func (d *Dispatcher) aggregateResults(results []NodeResult) []AggregatedGroup {
	logger.Infof("aggregateResults: 开始聚合结果, 结果数量: %d\n", len(results))

	groupsMap := make(map[string]*AggregatedGroup)

	for i, res := range results {
		logger.Infof("aggregateResults: 处理结果 [%d], 节点: %s, 状态: %s\n", i, res.NodeName, res.Status)

		// 计算指纹: Output + Error + Status
		// 简单起见，直接拼接字符串作为 Key
		key := d.calculateFingerprint(res)
		logger.Infof("aggregateResults: 计算指纹: %s\n", key)

		if _, exists := groupsMap[key]; !exists {
			logger.Infof("aggregateResults: 创建新组\n")
			groupsMap[key] = &AggregatedGroup{
				Output: res.Output,
				Error:  res.Error,
				Status: res.Status,
				Nodes:  []string{res.NodeName},
				Count:  1,
			}
		} else {
			logger.Infof("aggregateResults: 添加到现有组\n")
			groupsMap[key].Nodes = append(groupsMap[key].Nodes, res.NodeName)
			groupsMap[key].Count++
		}
	}

	// 将 Map 转换为 Slice
	var groups []AggregatedGroup
	for k, g := range groupsMap {
		logger.Infof("aggregateResults: 组 [%s], 节点数: %d\n", k, g.Count)
		groups = append(groups, *g)
	}

	logger.Infof("aggregateResults: 聚合完成, 组数: %d\n", len(groups))
	return groups
}

// calculateFingerprint 计算结果的指纹
func (d *Dispatcher) calculateFingerprint(res NodeResult) string {
	// 使用 SHA256 计算哈希
	h := sha256.New()
	h.Write([]byte(res.Output))
	h.Write([]byte(res.Error))
	h.Write([]byte(res.Status))
	return hex.EncodeToString(h.Sum(nil))
}
