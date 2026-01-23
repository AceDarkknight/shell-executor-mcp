package cmd

import (
	"fmt"
	"os"

	"shell-executor-mcp/internal/logger"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "k8s-mcp-client",
	Short: "Shell Executor MCP Client",
	Long:  `Shell Executor MCP Client is a tool that connects to the Shell Executor MCP Server and allows executing shell commands on a cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 调用 run 命令的逻辑
		RunCmd.Run(cmd, args)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is client_config.json)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringP("server", "s", "", "Server address")
	rootCmd.Flags().String("token", "", "Connection token")
	rootCmd.Flags().Bool("insecure-skip-verify", false, "Skip TLS verification")
	rootCmd.Flags().String("log-dir", "", "Log directory")
	rootCmd.Flags().StringP("log-level", "l", "info", "Log level (debug, info, warn, error)")

	// Bind flags to viper
	// 环境变量前缀为 MCP_CLIENT_
	viper.BindPFlag("server", rootCmd.Flags().Lookup("server"))
	viper.BindPFlag("token", rootCmd.Flags().Lookup("token"))
	viper.BindPFlag("insecure_skip_verify", rootCmd.Flags().Lookup("insecure-skip-verify"))
	viper.BindPFlag("log_dir", rootCmd.Flags().Lookup("log-dir"))
	viper.BindPFlag("log_level", rootCmd.Flags().Lookup("log-level"))

	// 设置环境变量前缀
	viper.SetEnvPrefix("MCP_CLIENT")
	viper.AutomaticEnv()

	// 添加子命令
	rootCmd.AddCommand(RunCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in home directory with name ".k8s-mcp-client" (without extension).
		viper.AddConfigPath(".")
		viper.SetConfigName("client_config")
		viper.SetConfigType("json")
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// 先初始化logger，避免死锁
		logger.InitLogger(nil, "client.log")
		logger.Infof("Using config file: %s", viper.ConfigFileUsed())
	} else {
		fmt.Printf("Failed to read config file: %v\n", err)
		os.Exit(1)
	}
}
