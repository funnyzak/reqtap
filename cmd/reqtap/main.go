package main

import (
	"fmt"
	"os"

	"github.com/funnyzak/reqtap/internal/config"
	"github.com/funnyzak/reqtap/internal/logger"
	"github.com/funnyzak/reqtap/internal/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "reqtap",
	Short: "Real-time, easy-to-use, highly customizable HTTP request debugging tool",
	Long: `ReqTap is a cross-platform, zero-dependency command-line tool for capturing, inspecting, and forwarding HTTP requests.

It can serve as a "request blackhole" or "webhook debugger" to help developers debug various HTTP requests.
`,
	RunE: runServer,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run:   showVersion,
}

func init() {
	// Add global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "Configuration file path")
	rootCmd.PersistentFlags().IntP("port", "p", 0, "Listen port")
	rootCmd.PersistentFlags().String("path", "", "URL path prefix to listen")
	rootCmd.PersistentFlags().StringP("log-level", "l", "", "Log level (trace, debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().Bool("log-file-enable", false, "Enable file logging")
	rootCmd.PersistentFlags().String("log-file-path", "", "Log file path")
	rootCmd.PersistentFlags().Int("log-file-max-size", 0, "Maximum size of a single log file (MB)")
	rootCmd.PersistentFlags().Int("log-file-max-backups", 0, "Maximum number of old log files to retain")
	rootCmd.PersistentFlags().Int("log-file-max-age", 0, "Maximum retention days for old log files")
	rootCmd.PersistentFlags().Bool("log-file-compress", false, "Whether to compress old log files")
	rootCmd.PersistentFlags().StringSliceP("forward-url", "f", []string{}, "Target URLs to forward")

	bindFlags(rootCmd)

	rootCmd.AddCommand(versionCmd)
}

func bindFlags(cmd *cobra.Command) {
	viper.BindPFlag("server.port", cmd.Flags().Lookup("port"))
	viper.BindPFlag("server.path", cmd.Flags().Lookup("path"))
	viper.BindPFlag("log.level", cmd.Flags().Lookup("log-level"))
	viper.BindPFlag("log.file_logging.enable", cmd.Flags().Lookup("log-file-enable"))
	viper.BindPFlag("log.file_logging.path", cmd.Flags().Lookup("log-file-path"))
	viper.BindPFlag("log.file_logging.max_size_mb", cmd.Flags().Lookup("log-file-max-size"))
	viper.BindPFlag("log.file_logging.max_backups", cmd.Flags().Lookup("log-file-max-backups"))
	viper.BindPFlag("log.file_logging.max_age_days", cmd.Flags().Lookup("log-file-max-age"))
	viper.BindPFlag("log.file_logging.compress", cmd.Flags().Lookup("log-file-compress"))
	viper.BindPFlag("forward.urls", cmd.Flags().Lookup("forward-url"))
}

func runServer(cmd *cobra.Command, args []string) error {
	// Get configuration file path
	configPath, _ := cmd.Flags().GetString("config")

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Create logger
	log := logger.NewLogger(&cfg.Log)

	// Display startup information
	printStartupBanner(cfg, log)

	// Create and start server
	srv := server.New(cfg, log)
	return srv.Start()
}

func showVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("ReqTap version %s\n", version)
	fmt.Printf("Commit: %s\n", commit)
	fmt.Printf("Built: %s\n", buildDate)
}

func printStartupBanner(cfg *config.Config, log logger.Logger) {
	fmt.Println()
	fmt.Println("══════════════════════════════════════════════════════════════════════")
	fmt.Println("                              ReqTap")
	fmt.Printf("                         HTTP Request Debugging Tool v%s\n", version)
	fmt.Println("══════════════════════════════════════════════════════════════════════")
	fmt.Printf("Listen Address: http://0.0.0.0:%d%s\n", cfg.Server.Port, cfg.Server.Path)
	if len(cfg.Forward.URLs) > 0 {
		fmt.Printf("Forward Targets: %v\n", cfg.Forward.URLs)
	}
	if cfg.Log.FileLogging.Enable {
		fmt.Printf("Log File: %s\n", cfg.Log.FileLogging.Path)
	}
	fmt.Println("══════════════════════════════════════════════════════════════════════")
	fmt.Println()

	log.Info("ReqTap starting",
		"version", version,
		"port", cfg.Server.Port,
		"path", cfg.Server.Path,
		"log_level", cfg.Log.Level,
		"forward_urls", cfg.Forward.URLs,
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
