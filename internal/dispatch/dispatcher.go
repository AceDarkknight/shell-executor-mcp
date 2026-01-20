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
	// 用于收集所有结果
	var results []NodeResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 1. 本地执行
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := localExecutor.Execute(cmd, 5*time.Second)
		if err != nil {
			mu.Lock()
			results = append(results, NodeResult{
				NodeName: nodeName,
				Status:   "failed",
				Error:    err.Error(),
			})
			mu.Unlock()
			return
		}

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
	for _, peer := range d.peers {
		wg.Add(1)
		go func(peerURL string) {
			defer wg.Done()
			result := d.executeOnPeer(peerURL, cmd)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(peer)
	}

	// 等待所有任务完成
	wg.Wait()

	// 3. 聚合结果
	groups := d.aggregateResults(results)
	summary := fmt.Sprintf("Executed on %d nodes, %d groups found", len(results), len(groups))

	return groups, summary
}

// executeOnPeer 在指定的 Peer 节点上执行命令
func (d *Dispatcher) executeOnPeer(peerURL string, cmd string) NodeResult {
	reqBody := DispatchRequest{Cmd: cmd}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return NodeResult{
			NodeName: peerURL,
			Status:   "failed",
			Error:    fmt.Sprintf("marshal request failed: %v", err),
		}
	}

	// 构建请求 URL
	// peerURL 应该是完整的 http://host:port
	url := fmt.Sprintf("%s/internal/exec", peerURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return NodeResult{
			NodeName: peerURL,
			Status:   "failed",
			Error:    fmt.Sprintf("create request failed: %v", err),
		}
	}

	req.Header.Set("Content-Type", "application/json")
	if d.token != "" {
		req.Header.Set("X-Cluster-Token", d.token)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return NodeResult{
			NodeName: peerURL,
			Status:   "failed",
			Error:    fmt.Sprintf("request failed: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return NodeResult{
			NodeName: peerURL,
			Status:   "failed",
			Error:    fmt.Sprintf("server returned %d: %s", resp.StatusCode, string(body)),
		}
	}

	var respData DispatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return NodeResult{
			NodeName: peerURL,
			Status:   "failed",
			Error:    fmt.Sprintf("decode response failed: %v", err),
		}
	}

	status := "success"
	if respData.ExitCode != 0 || respData.Error != "" {
		status = "failed"
	}

	return NodeResult{
		NodeName: peerURL,
		Status:   status,
		Output:   respData.Output,
		Error:    respData.Error,
	}
}

// aggregateResults 将结果按输出内容进行分组压缩
func (d *Dispatcher) aggregateResults(results []NodeResult) []AggregatedGroup {
	groupsMap := make(map[string]*AggregatedGroup)

	for _, res := range results {
		// 计算指纹: Output + Error + Status
		// 简单起见，直接拼接字符串作为 Key
		key := d.calculateFingerprint(res)

		if _, exists := groupsMap[key]; !exists {
			groupsMap[key] = &AggregatedGroup{
				Output: res.Output,
				Error:  res.Error,
				Status: res.Status,
				Nodes:  []string{res.NodeName},
				Count:  1,
			}
		} else {
			groupsMap[key].Nodes = append(groupsMap[key].Nodes, res.NodeName)
			groupsMap[key].Count++
		}
	}

	// 将 Map 转换为 Slice
	var groups []AggregatedGroup
	for _, g := range groupsMap {
		groups = append(groups, *g)
	}

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
