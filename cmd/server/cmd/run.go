package cmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/AceDarkknight/shell-executor-mcp/internal/security"

	"github.com/AceDarkknight/shell-executor-mcp/internal/logger"

	"github.com/AceDarkknight/shell-executor-mcp/internal/executor"

	"github.com/AceDarkknight/shell-executor-mcp/internal/dispatch"

	"github.com/AceDarkknight/shell-executor-mcp/internal/config"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RunCmd 表示 run 命令
var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "启动 MCP 服务器",
	Long:  `启动 MCP 服务器并开始监听请求。`,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

// runServer 启动 MCP 服务器
func runServer() {
	// 1. 加载配置
	// 如果没有指定配置文件，尝试从 viper 读取
	var cfg *config.ServerConfig
	var err error

	if cfgFile != "" {
		// 使用指定的配置文件
		cfg, err = config.LoadServerConfig(cfgFile)
		if err != nil {
			logger.Fatalf("Failed to load config: %v", err)
		}
		logger.Infof("Loaded config from file: %+v", cfg)
	} else {
		// 从 viper 读取配置（可能来自环境变量或默认配置文件）
		cfg, err = loadConfigFromViper()
		if err != nil {
			logger.Fatalf("Failed to load config from viper: %v", err)
		}
		logger.Infof("成功从 viper 加载配置")
	}

	// 初始化日志（必须在调用任何logger函数之前）
	logCfg := &cfg.LogConfig
	if err := logger.InitLogger(logCfg, "server.log"); err != nil {
		logger.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Debugf("开始启动 MCP 服务器")

	if cfgFile != "" {
		logger.Debugf("使用指定的配置文件: %s", cfgFile)
		logger.Infof("成功加载配置文件: %s", cfgFile)
	} else {
		logger.Debugf("未指定配置文件，从 viper 读取配置")
		logger.Infof("成功从 viper 加载配置")
	}

	logger.Infof("========================================")
	logger.Infof("Server starting as node: %s", cfg.NodeName)
	logger.Infof("Listening on port: %d", cfg.Port)
	logger.Infof("========================================")

	// 2. 初始化组件
	logger.Debugf("初始化安全卫士，黑名单命令: %v", cfg.Security.BlacklistedCommands)
	logger.Debugf("初始化安全卫士，危险参数正则: %v", cfg.Security.DangerousArgsRegex)
	guard, err := security.NewGuard(cfg.Security.BlacklistedCommands, cfg.Security.DangerousArgsRegex)
	if err != nil {
		logger.Fatalf("Failed to initialize security guard: %v", err)
	}
	logger.Infof("安全卫士初始化成功")

	logger.Debugf("初始化命令执行器")
	executor := executor.NewExecutor()
	logger.Infof("命令执行器初始化成功")

	logger.Debugf("初始化集群分发器，peers: %v, token: %s", cfg.GetPeers(), cfg.ClusterToken)
	dispatcher := dispatch.NewDispatcher(cfg.GetPeers(), cfg.ClusterToken)
	logger.Infof("集群分发器初始化成功")

	// 3. 创建 MCP Server
	logger.Debugf("创建 MCP Server: name=shell-executor-mcp, version=1.0.0")
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "shell-executor-mcp",
		Version: "1.0.0",
	}, nil)
	logger.Infof("MCP Server 创建成功")

	// 4. 注册 MCP Tools
	logger.Debugf("注册 MCP Tools")
	registerTools(mcpServer, guard, executor, dispatcher, cfg)
	logger.Infof("MCP Tools 注册成功")

	// 5. 创建 HTTP Handler (Streamable HTTP)
	// 使用 StreamableHTTPHandler 提供 MCP Streamable HTTP endpoint
	logger.Debugf("创建 StreamableHTTPHandler，stateless=true, jsonResponse=true")
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})
	logger.Infof("HTTP Handler 创建成功")

	// 6. 注册内部 API 端点
	// 我们需要将内部 API 挂载到同一个 http.ServeMux 上
	// 但 mcpHandler 本身是一个 http.Handler
	// 我们使用 http.NewServeMux 并将 MCP handler 挂载到 /mcp，内部 API 挂载到 /internal
	logger.Debugf("创建 HTTP ServeMux 并注册路由")
	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpHandler)
	logger.Debugf("注册 MCP handler 到 /mcp")

	// 包装内部 API Handler 以确保它们可以被访问
	mux.HandleFunc("/internal/exec", internalExecHandler(guard, executor, cfg.ClusterToken))
	logger.Debugf("注册内部 API: /internal/exec")

	// 健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	logger.Debugf("注册健康检查: /health")

	mux.HandleFunc("/internal/join", internalJoinHandler(cfg, cfgFile))
	logger.Debugf("注册内部 API: /internal/join")
	mux.HandleFunc("/internal/sync", internalSyncHandler(cfg, cfgFile))
	logger.Debugf("注册内部 API: /internal/sync")

	// 7. 启动 HTTP Server
	addr := ":" + strconv.Itoa(cfg.Port)
	logger.Infof("========================================")
	if cfg.TLS.Enabled {
		logger.Infof("Server listening on %s (HTTPS/TLS)", addr)
		logger.Infof("MCP endpoint: https://localhost%s/mcp", addr)
	} else {
		logger.Infof("Server listening on %s", addr)
		logger.Infof("MCP endpoint: http://localhost%s/mcp", addr)
	}
	logger.Infof("Internal API endpoints: /internal/...")
	logger.Infof("========================================")
	logger.Infof("服务器启动完成，等待请求...")

	if cfg.TLS.Enabled {
		tlsConfig, err := buildTLSConfig(cfg)
		if err != nil {
			logger.Fatalf("Failed to build TLS config: %v", err)
		}
		server := &http.Server{
			Addr:      addr,
			Handler:   mux,
			TLSConfig: tlsConfig,
		}
		// 如果使用自动生成的证书，TLSConfig 已包含证书，传空字符串
		if err := server.ListenAndServeTLS("", ""); err != nil {
			logger.Fatalf("HTTPS Server failed: %v", err)
		}
	} else {
		if err := http.ListenAndServe(addr, mux); err != nil {
			logger.Fatalf("Server failed: %v", err)
		}
	}
}

// loadConfigFromViper 从 viper 加载配置
func loadConfigFromViper() (*config.ServerConfig, error) {
	cfg := &config.ServerConfig{
		Port:         viper.GetInt("port"),
		NodeName:     viper.GetString("node_name"),
		ClusterToken: viper.GetString("token"),
		Security: config.SecurityConfig{
			BlacklistedCommands: []string{},
			DangerousArgsRegex:  []string{},
		},
		LogConfig: logger.LogConfig{
			Level:      viper.GetString("log_level"),
			LogDir:     viper.GetString("log_dir"),
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		},
	}

	// 如果 log_dir 为空，使用默认值
	if cfg.LogConfig.LogDir == "" {
		cfg.LogConfig.LogDir = "logs"
	}

	// 如果 node_name 为空，使用 hostname
	if cfg.NodeName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("failed to get hostname: %v", err)
		}
		cfg.NodeName = hostname
	}

	// 尝试从 viper 读取 peers
	peers := viper.GetStringSlice("peers")
	cfg.Peers = peers

	// TLS 配置
	cfg.TLS = config.TLSConfig{
		Enabled:  viper.GetBool("tls_enabled"),
		CertFile: viper.GetString("tls_cert"),
		KeyFile:  viper.GetString("tls_key"),
	}

	return cfg, nil
}

// internalExecHandler 处理内部执行请求 (Server -> Server)
func internalExecHandler(guard *security.Guard, executor *executor.Executor, clusterToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debugf("收到 /internal/exec 请求，方法: %s, 远程地址: %s", r.Method, r.RemoteAddr)

		if r.Method != http.MethodPost {
			logger.Warnf("Invalid method for /internal/exec: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Token 验证
		if clusterToken != "" {
			token := r.Header.Get("X-Cluster-Token")
			if token != clusterToken {
				logger.Warnf("Cluster Token 校验失败, remote=%s, token=%s", r.RemoteAddr, token)
				http.Error(w, "Unauthorized: invalid cluster token", http.StatusUnauthorized)
				return
			}
			logger.Debugf("Cluster Token 校验通过")
		}

		var req dispatch.DispatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Errorf("Failed to decode request: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		logger.Infof("收到内部执行请求，命令: %s", req.Cmd)

		// 安全检查
		logger.Debugf("开始安全检查")
		if err := guard.CheckCommand(req.Cmd); err != nil {
			logger.Warnf("安全检查失败，命令被拦截: %s, 错误: %v", req.Cmd, err)
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		logger.Debugf("安全检查通过")

		// 执行
		logger.Debugf("开始执行命令，超时: 5s")
		result, err := executor.Execute(req.Cmd, 5*time.Second)
		if err != nil {
			logger.Errorf("命令执行失败: %v", err)
			// 即使有错误，也返回部分结果
			// result.Error 已经包含了错误信息
		} else {
			logger.Infof("命令执行成功，退出码: %d, 输出长度: %d", result.ExitCode, len(result.Output))
		}

		logger.Debugf("返回执行结果")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// internalJoinHandler 处理节点加入请求
func internalJoinHandler(cfg *config.ServerConfig, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debugf("收到 /internal/join 请求，方法: %s, 远程地址: %s", r.Method, r.RemoteAddr)

		if r.Method != http.MethodPost {
			logger.Warnf("Invalid method for /internal/join: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			MyAddr string `json:"my_addr"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Errorf("Failed to decode join request: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		logger.Infof("收到节点加入请求，地址: %s", req.MyAddr)

		// 添加新节点
		logger.Debugf("添加新节点到 peers 列表")
		cfg.AddPeer(req.MyAddr)
		logger.Infof("当前 peers 数量: %d", len(cfg.GetPeers()))

		// 广播给其他节点 (异步)
		logger.Debugf("开始广播同步到其他节点")
		go broadcastSync(cfg, configPath)

		// 返回当前所有节点
		logger.Debugf("返回当前所有 peers")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"peers": cfg.GetPeers(),
		})
	}
}

// internalSyncHandler 处理同步节点列表请求
func internalSyncHandler(cfg *config.ServerConfig, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debugf("收到 /internal/sync 请求，方法: %s, 远程地址: %s", r.Method, r.RemoteAddr)

		if r.Method != http.MethodPost {
			logger.Warnf("Invalid method for /internal/sync: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Peers []string `json:"peers"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Errorf("Failed to decode sync request: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		logger.Infof("收到同步请求，peers: %v", req.Peers)

		// 更新本地 Peers
		logger.Debugf("更新本地 peers 列表")
		cfg.SetPeers(req.Peers)
		logger.Infof("本地 peers 更新完成，数量: %d", len(cfg.GetPeers()))

		// 持久化
		// 注意：这里需要知道配置文件路径，简化起见暂时跳过
		// cfg.Save(configPath)
		logger.Debugf("持久化配置（当前跳过）")

		w.WriteHeader(http.StatusOK)
	}
}

// broadcastSync 将当前的 Peer 列表广播给所有已知节点
func broadcastSync(cfg *config.ServerConfig, configPath string) {
	peers := cfg.GetPeers()
	logger.Infof("开始广播同步到 %d 个 peers", len(peers))
	// 这里应该使用 HTTP Client 发送 POST /internal/sync
	// 简化实现，略
	logger.Debugf("广播同步完成（当前为简化实现）")
}

// buildTLSConfig 构建 TLS 配置
func buildTLSConfig(cfg *config.ServerConfig) (*tls.Config, error) {
	if cfg.TLS.CertFile != "" && cfg.TLS.KeyFile != "" {
		// 使用用户提供的证书
		cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load cert/key: %v", err)
		}
		logger.Infof("使用指定证书: cert=%s, key=%s", cfg.TLS.CertFile, cfg.TLS.KeyFile)
		return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
	}

	// 自动生成自签证书
	logger.Infof("未指定证书文件，自动生成自签证书...")
	cert, err := generateSelfSignedCert()
	if err != nil {
		return nil, fmt.Errorf("failed to generate self-signed cert: %v", err)
	}
	logger.Infof("自签证书生成成功")
	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}

// generateSelfSignedCert 生成内存中的自签名 TLS 证书
func generateSelfSignedCert() (tls.Certificate, error) {
	// 生成 ECDSA P-256 私钥
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	// 创建证书模板
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Shell Executor MCP"},
			CommonName:   "shell-executor-mcp",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour), // 10年
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		// 添加 SAN，支持 IP 和 localhost 访问
		IPAddresses: []net.IP{net.ParseIP("0.0.0.0"), net.ParseIP("127.0.0.1")},
		DNSNames:    []string{"localhost", "*"},
	}

	// 自签名
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	// 编码为 PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return tls.X509KeyPair(certPEM, keyPEM)
}
