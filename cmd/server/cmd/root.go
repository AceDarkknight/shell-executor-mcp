package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "k8s-mcp-server",
	Short: "Shell Executor MCP Server",
	Long:  `Shell Executor MCP Server is a tool that allows executing shell commands on a cluster via the Model Context Protocol.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 调用 run 命令的逻辑
		RunCmd.Run(cmd, args)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is server_config.json)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().IntP("port", "p", 8080, "Server listening port")
	rootCmd.Flags().String("cert", "", "TLS certificate file path")
	rootCmd.Flags().String("key", "", "TLS key file path")
	rootCmd.Flags().Bool("insecure", false, "Use insecure connection (default false)")
	rootCmd.Flags().String("token", "", "Security token")
	rootCmd.Flags().String("log-dir", "", "Log directory")
	rootCmd.Flags().StringP("node-name", "n", "", "Node name (default to hostname)")
	rootCmd.Flags().StringP("log-level", "l", "info", "Log level (debug, info, warn, error)")

	// Bind flags to viper
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

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in home directory with name ".k8s-mcp-server" (without extension).
		viper.AddConfigPath(".")
		viper.SetConfigName("server_config")
		viper.SetConfigType("json")
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
