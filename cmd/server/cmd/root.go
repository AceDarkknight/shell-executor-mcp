package cmd

import (
	"os"
	"shell-executor-mcp/internal/logger"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd 表示不使用子命令时的基础命令
var rootCmd = &cobra.Command{
	Use:   "k8s-mcp-server",
	Short: "Shell Executor MCP 服务器",
	Long:  `Shell Executor MCP Server 是一个允许通过 Model Context Protocol 在集群上执行 shell 命令的工具。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 调用 run 命令的逻辑
		RunCmd.Run(cmd, args)
	},
}

// Execute 将所有子命令添加到根命令并设置适当的标志。
// 这由 main.main()调用。只需对 rootCmd执行一次。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// 在这里定义标志和配置设置。
	// Cobra支持持久标志，如果在这里定义，
	// 将对整个应用程序全局有效。

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is server_config.json)")

	// Cobra 还支持本地标志，这些标志
	// 仅在直接调用此操作时运行。
	rootCmd.Flags().IntP("port", "p", 8080, "Server listening port")
	rootCmd.Flags().String("cert", "", "TLS certificate file path")
	rootCmd.Flags().String("key", "", "TLS key file path")
	rootCmd.Flags().Bool("insecure", false, "Use insecure connection (default false)")
	rootCmd.Flags().String("token", "", "Security token")
	rootCmd.Flags().String("log-dir", "", "Log directory")
	rootCmd.Flags().StringP("node-name", "n", "", "Node name (default to hostname)")
	rootCmd.Flags().StringP("log-level", "l", "info", "Log level (debug, info, warn, error)")

	// 将标志绑定到 viper
	// 环境变量前缀为 MCP_
	viper.BindPFlag("port", rootCmd.Flags().Lookup("port"))
	viper.BindPFlag("cert", rootCmd.Flags().Lookup("cert"))
	viper.BindPFlag("key", rootCmd.Flags().Lookup("key"))
	viper.BindPFlag("insecure", rootCmd.Flags().Lookup("insecure"))
	viper.BindPFlag("token", rootCmd.Flags().Lookup("token"))
	viper.BindPFlag("log_dir", rootCmd.Flags().Lookup("log-dir"))
	viper.BindPFlag("node_name", rootCmd.Flags().Lookup("node-name"))
	viper.BindPFlag("log_level", rootCmd.Flags().Lookup("log-level"))

	// 设置环境变量前缀
	viper.SetEnvPrefix("MCP")
	viper.AutomaticEnv()

	// 添加子命令
	rootCmd.AddCommand(RunCmd)
}

// initConfig 读取配置文件和环境变量（如果已设置）。
func initConfig() {
	if cfgFile != "" {
		// 使用标志指定的配置文件。
		viper.SetConfigFile(cfgFile)
	} else {
		// 在当前目录中搜索名为 ".k8s-mcp-server" 的配置文件（不带扩展名）。
		viper.AddConfigPath(".")
		viper.SetConfigName("server_config")
		viper.SetConfigType("json")
	}

	// 如果找到配置文件，则读取它。
	if err := viper.ReadInConfig(); err == nil {
		// 先初始化logger，避免死锁
		logger.InitLogger(nil, "server.log")
		logger.Infof("Using config file: %s", viper.ConfigFileUsed())
	}
}
