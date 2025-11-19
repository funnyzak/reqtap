package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
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

var examplesCmd = &cobra.Command{
	Use:   "examples",
	Short: "Show common usage examples",
	Long: `Display common usage scenarios and example commands for ReqTap.

This includes examples for:
- Basic request debugging
- Webhook testing
- Request forwarding
- Web console usage
- Configuration file usage
- Production deployment
`,
	Run: showExamples,
}

func init() {
	// Add global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "Configuration file path")
	rootCmd.PersistentFlags().IntP("port", "p", 0, "Listen port")
	rootCmd.PersistentFlags().String("path", "", "URL path prefix to listen")
	rootCmd.PersistentFlags().Int64("max-body-bytes", 0, "Maximum request body size in bytes (0 for unlimited)")
	rootCmd.PersistentFlags().StringP("log-level", "l", "", "Log level (trace, debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().Bool("log-file-enable", false, "Enable file logging")
	rootCmd.PersistentFlags().String("log-file-path", "", "Log file path")
	rootCmd.PersistentFlags().Int("log-file-max-size", 0, "Maximum size of a single log file (MB)")
	rootCmd.PersistentFlags().Int("log-file-max-backups", 0, "Maximum number of old log files to retain")
	rootCmd.PersistentFlags().Int("log-file-max-age", 0, "Maximum retention days for old log files")
	rootCmd.PersistentFlags().Bool("log-file-compress", false, "Whether to compress old log files")
	rootCmd.PersistentFlags().StringSliceP("forward-url", "f", []string{}, "Target URLs to forward")
	rootCmd.PersistentFlags().Bool("silence", false, "Suppress interactive console output")
	rootCmd.PersistentFlags().Bool("json", false, "Emit structured JSON output")
	rootCmd.PersistentFlags().Bool("body-view", false, "Enable structured body formatting in console mode")
	rootCmd.PersistentFlags().Int("body-preview-bytes", 0, "Maximum bytes to preview before truncating console body output")
	rootCmd.PersistentFlags().Bool("full-body", false, "Always print full request bodies, ignoring preview limits")
	rootCmd.PersistentFlags().Bool("body-hex-preview", false, "Enable hexadecimal preview for binary bodies")
	rootCmd.PersistentFlags().Int("body-hex-preview-bytes", 0, "Limit for hexadecimal preview bytes (0 keeps config value)")
	rootCmd.PersistentFlags().Bool("body-save-binary", false, "Persist binary bodies to disk when enabled")
	rootCmd.PersistentFlags().String("body-save-directory", "", "Directory to persist binary bodies (requires --body-save-binary)")

	// Web console configuration flags
	rootCmd.PersistentFlags().Bool("web-enable", false, "Enable/disable web console")
	rootCmd.PersistentFlags().String("web-path", "", "Web UI access path")
	rootCmd.PersistentFlags().String("web-admin-path", "", "Web admin API path")
	rootCmd.PersistentFlags().Int("web-max-requests", 0, "Maximum number of requests to retain in memory")
	rootCmd.PersistentFlags().Bool("web-auth-enable", false, "Enable/disable web console authentication")
	rootCmd.PersistentFlags().String("web-auth-session-timeout", "", "Web console session timeout duration")
	rootCmd.PersistentFlags().Bool("web-export-enable", false, "Enable/disable web console data export")
	rootCmd.PersistentFlags().StringSlice("web-export-formats", []string{}, "Supported export formats for web console")

	bindFlags(rootCmd)

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(examplesCmd)
}

func bindFlags(cmd *cobra.Command) {
	viper.BindPFlag("server.port", cmd.Flags().Lookup("port"))
	viper.BindPFlag("server.path", cmd.Flags().Lookup("path"))
	viper.BindPFlag("server.max_body_bytes", cmd.Flags().Lookup("max-body-bytes"))
	viper.BindPFlag("log.level", cmd.Flags().Lookup("log-level"))
	viper.BindPFlag("log.file_logging.enable", cmd.Flags().Lookup("log-file-enable"))
	viper.BindPFlag("log.file_logging.path", cmd.Flags().Lookup("log-file-path"))
	viper.BindPFlag("log.file_logging.max_size_mb", cmd.Flags().Lookup("log-file-max-size"))
	viper.BindPFlag("log.file_logging.max_backups", cmd.Flags().Lookup("log-file-max-backups"))
	viper.BindPFlag("log.file_logging.max_age_days", cmd.Flags().Lookup("log-file-max-age"))
	viper.BindPFlag("log.file_logging.compress", cmd.Flags().Lookup("log-file-compress"))
	viper.BindPFlag("forward.urls", cmd.Flags().Lookup("forward-url"))

	// Web console configuration bindings
	viper.BindPFlag("web.enable", cmd.Flags().Lookup("web-enable"))
	viper.BindPFlag("web.path", cmd.Flags().Lookup("web-path"))
	viper.BindPFlag("web.admin_path", cmd.Flags().Lookup("web-admin-path"))
	viper.BindPFlag("web.max_requests", cmd.Flags().Lookup("web-max-requests"))
	viper.BindPFlag("web.auth.enable", cmd.Flags().Lookup("web-auth-enable"))
	viper.BindPFlag("web.auth.session_timeout", cmd.Flags().Lookup("web-auth-session-timeout"))
	viper.BindPFlag("web.export.enable", cmd.Flags().Lookup("web-export-enable"))
	viper.BindPFlag("web.export.formats", cmd.Flags().Lookup("web-export-formats"))
	viper.BindPFlag("output.silence", cmd.Flags().Lookup("silence"))
	viper.BindPFlag("output.body_view.enable", cmd.Flags().Lookup("body-view"))
	viper.BindPFlag("output.body_view.max_preview_bytes", cmd.Flags().Lookup("body-preview-bytes"))
	viper.BindPFlag("output.body_view.full_body", cmd.Flags().Lookup("full-body"))
	viper.BindPFlag("output.body_view.binary.hex_preview_enable", cmd.Flags().Lookup("body-hex-preview"))
	viper.BindPFlag("output.body_view.binary.hex_preview_bytes", cmd.Flags().Lookup("body-hex-preview-bytes"))
	viper.BindPFlag("output.body_view.binary.save_to_file", cmd.Flags().Lookup("body-save-binary"))
	viper.BindPFlag("output.body_view.binary.save_directory", cmd.Flags().Lookup("body-save-directory"))
}

func runServer(cmd *cobra.Command, args []string) error {
	// Get configuration file path
	configPath, _ := cmd.Flags().GetString("config")

	// Load configuration using global viper
	cfg, err := config.LoadConfig(configPath, viper.GetViper())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override with command line arguments (command line has highest priority)
	// This ensures command line flags override config file values
	if port, err := cmd.Flags().GetInt("port"); err == nil && port != 0 {
		cfg.Server.Port = port
	}
	if path, err := cmd.Flags().GetString("path"); err == nil && path != "" {
		cfg.Server.Path = path
	}
	if cmd.Flags().Changed("max-body-bytes") {
		if maxBodyBytes, err := cmd.Flags().GetInt64("max-body-bytes"); err == nil {
			cfg.Server.MaxBodyBytes = maxBodyBytes
		}
	}
	if logLevel, err := cmd.Flags().GetString("log-level"); err == nil && logLevel != "" {
		cfg.Log.Level = logLevel
	}
	if logFileEnable, err := cmd.Flags().GetBool("log-file-enable"); err == nil && cmd.Flags().Changed("log-file-enable") {
		cfg.Log.FileLogging.Enable = logFileEnable
	}
	if logFilePath, err := cmd.Flags().GetString("log-file-path"); err == nil && logFilePath != "" {
		cfg.Log.FileLogging.Path = logFilePath
	}
	if logFileSize, err := cmd.Flags().GetInt("log-file-max-size"); err == nil && logFileSize != 0 {
		cfg.Log.FileLogging.MaxSizeMB = logFileSize
	}
	if logFileBackups, err := cmd.Flags().GetInt("log-file-max-backups"); err == nil && logFileBackups != 0 {
		cfg.Log.FileLogging.MaxBackups = logFileBackups
	}
	if logFileAge, err := cmd.Flags().GetInt("log-file-max-age"); err == nil && logFileAge != 0 {
		cfg.Log.FileLogging.MaxAgeDays = logFileAge
	}
	if logFileCompress, err := cmd.Flags().GetBool("log-file-compress"); err == nil && cmd.Flags().Changed("log-file-compress") {
		cfg.Log.FileLogging.Compress = logFileCompress
	}
	if forwardURLs, err := cmd.Flags().GetStringSlice("forward-url"); err == nil && len(forwardURLs) > 0 {
		cfg.Forward.URLs = forwardURLs
	}

	// Override with web console command line arguments (command line has highest priority)
	if webEnable, err := cmd.Flags().GetBool("web-enable"); err == nil && cmd.Flags().Changed("web-enable") {
		cfg.Web.Enable = webEnable
	}
	if webPath, err := cmd.Flags().GetString("web-path"); err == nil && webPath != "" {
		cfg.Web.Path = webPath
	}
	if webAdminPath, err := cmd.Flags().GetString("web-admin-path"); err == nil && webAdminPath != "" {
		cfg.Web.AdminPath = webAdminPath
	}
	if webMaxRequests, err := cmd.Flags().GetInt("web-max-requests"); err == nil && webMaxRequests != 0 {
		cfg.Web.MaxRequests = webMaxRequests
	}
	if webAuthEnable, err := cmd.Flags().GetBool("web-auth-enable"); err == nil && cmd.Flags().Changed("web-auth-enable") {
		cfg.Web.Auth.Enable = webAuthEnable
	}
	if webAuthSessionTimeout, err := cmd.Flags().GetString("web-auth-session-timeout"); err == nil && webAuthSessionTimeout != "" {
		if timeout, err := time.ParseDuration(webAuthSessionTimeout); err == nil {
			cfg.Web.Auth.SessionTimeout = timeout
		}
	}
	if webExportEnable, err := cmd.Flags().GetBool("web-export-enable"); err == nil && cmd.Flags().Changed("web-export-enable") {
		cfg.Web.Export.Enable = webExportEnable
	}
	if webExportFormats, err := cmd.Flags().GetStringSlice("web-export-formats"); err == nil && len(webExportFormats) > 0 {
		cfg.Web.Export.Formats = webExportFormats
	}

	if cmd.Flags().Changed("silence") {
		if silence, err := cmd.Flags().GetBool("silence"); err == nil {
			cfg.Output.Silence = silence
		}
	}
	if jsonOutput, err := cmd.Flags().GetBool("json"); err == nil && jsonOutput {
		cfg.Output.Mode = "json"
	}
	if cmd.Flags().Changed("body-view") {
		if bodyView, err := cmd.Flags().GetBool("body-view"); err == nil {
			cfg.Output.BodyView.Enable = bodyView
		}
	}
	if cmd.Flags().Changed("body-preview-bytes") {
		if preview, err := cmd.Flags().GetInt("body-preview-bytes"); err == nil {
			cfg.Output.BodyView.MaxPreviewBytes = preview
		}
	}
	if cmd.Flags().Changed("full-body") {
		if fullBody, err := cmd.Flags().GetBool("full-body"); err == nil {
			cfg.Output.BodyView.FullBody = fullBody
		}
	}
	if cmd.Flags().Changed("body-hex-preview") {
		if hexPreview, err := cmd.Flags().GetBool("body-hex-preview"); err == nil {
			cfg.Output.BodyView.Binary.HexPreviewEnable = hexPreview
		}
	}
	if cmd.Flags().Changed("body-hex-preview-bytes") {
		if bytes, err := cmd.Flags().GetInt("body-hex-preview-bytes"); err == nil {
			cfg.Output.BodyView.Binary.HexPreviewBytes = bytes
		}
	}
	if cmd.Flags().Changed("body-save-binary") {
		if save, err := cmd.Flags().GetBool("body-save-binary"); err == nil {
			cfg.Output.BodyView.Binary.SaveToFile = save
		}
	}
	if cmd.Flags().Changed("body-save-directory") {
		if dir, err := cmd.Flags().GetString("body-save-directory"); err == nil {
			cfg.Output.BodyView.Binary.SaveDirectory = dir
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	if err := validateWebPathConflicts(cfg); err != nil {
		return err
	}

	// Create logger
	log := logger.NewLogger(&cfg.Log, cfg.Output.Mode)

	// Display startup information
	if !cfg.Output.Silence && strings.ToLower(cfg.Output.Mode) != "json" {
		printStartupBanner(cfg, log)
	}
	logStartupSummary(cfg, log)

	// Create and start server
	srv := server.New(cfg, log)
	return srv.Start()
}

func showVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("ReqTap version %s\n", version)
	fmt.Printf("Commit: %s\n", commit)
	fmt.Printf("Built: %s\n", buildDate)
}

func showExamples(cmd *cobra.Command, args []string) {
	examples := `ReqTap Usage Examples

Basic Usage
  # Start default server (port 8080, listen to all paths)
  reqtap

  # Specify port
  reqtap -p 3000

  # Specify listening path prefix
  reqtap --path /webhook

Webhook Debugging
  # Start webhook debugging server
  reqtap -p 8080 --path /webhook --log-level debug

  # Enable file logging
  reqtap -p 8080 --log-file-enable --log-file-path ./webhook.log

Request Forwarding
  # Forward to single target
  reqtap -p 8080 --forward-url http://localhost:3000/webhook

  # Forward to multiple targets
  reqtap -p 8080 --forward-url http://localhost:3000/api --forward-url http://localhost:4000/backup

  # Real-world: GitHub Webhook forwarding to multiple services
  reqtap -p 8080 --path /github --forward-url http://service-a/webhook --forward-url http://service-b/hook

Web Console
  # Enable Web Console
  reqtap -p 8080 --web-enable

  # Custom Web Console paths
  reqtap -p 8080 --web-enable --web-path /console --web-admin-path /api

  # Enable authentication and data export
  reqtap -p 8080 --web-enable --web-auth-enable --web-export-enable

Configuration File
  # Use configuration file
  reqtap -c config.yaml

  # Override config file with command line arguments
  reqtap -c config.yaml -p 9000 --log-level debug

Production Environment
  # Recommended production configuration
  reqtap -p 8080 \
    --log-level info \
    --log-file-enable \
    --log-file-path /var/log/reqtap.log \
    --log-file-max-size 100 \
    --log-file-max-backups 7 \
    --log-file-max-age 30 \
    --log-file-compress

Development Debugging
  # Development environment debugging mode
  reqtap -p 8080 \
    --log-level debug \
    --web-enable \
    --web-auth-enable \
    --web-export-enable

Common Scenario Examples

  1. API Development Debugging
     reqtap -p 8080 --path /api --forward-url http://backend:8000/api --web-enable

  2. Microservice Gateway
     reqtap -p 80 \
       --forward-url http://service-a:8080 \
       --forward-url http://service-b:8080 \
       --log-file-enable \
       --web-enable

  3. Webhook Testing
     reqtap -p 8080 --path /stripe --forward-url http://localhost:3000/stripe/webhook

  4. Load Balancing Testing
     reqtap -p 8080 \
       --forward-url http://server-1:3000 \
       --forward-url http://server-2:3000 \
       --forward-url http://server-3:3000

  5. Request Monitoring
     reqtap -p 80 --log-file-enable --web-enable --web-export-enable

Tips
  - Use 'reqtap version' to check version information
  - Use '--help' to see all available parameters
  - For more configuration options, refer to the configuration file documentation`

	fmt.Println(examples)
}

func printStartupBanner(cfg *config.Config, log logger.Logger) {
	// Collect all content lines to display
	var lines []string

	// Title lines
	titleLine := fmt.Sprintf("ReqTap v%s", version)
	subtitleLine := "Request Inspector & Forwarding Tool"

	// Empty line
	lines = append(lines, titleLine, subtitleLine, "")

	// Listening information
	watchPath := cfg.Server.Path
	if watchPath == "/" {
		watchPath = "/ (All Paths)"
	}
	lines = append(lines, fmt.Sprintf("🚀 Listening on:   http://0.0.0.0:%d%s", cfg.Server.Port, cfg.Server.Path))
	lines = append(lines, fmt.Sprintf("🎯 Watching Path:   %s", watchPath))
	lines = append(lines, fmt.Sprintf("📊 Log Level:       %s", cfg.Log.Level))
	if cfg.Web.Enable {
		lines = append(lines, "🖥️ Web Console:    Enabled")
		lines = append(lines, fmt.Sprintf("   └─ UI Path:      %s", cfg.Web.Path))
		lines = append(lines, fmt.Sprintf("   └─ API Path:     %s", cfg.Web.AdminPath))
		if cfg.Web.Auth.Enable {
			lines = append(lines, fmt.Sprintf("   └─ Auth:         Enabled (%d user(s))", len(cfg.Web.Auth.Users)))
			// Add user details
			for _, user := range cfg.Web.Auth.Users {
				lines = append(lines, fmt.Sprintf("      └─ User:      %s (%s) - %s", user.Username, user.Role, user.Password))
			}
			// Add session timeout info
			lines = append(lines, fmt.Sprintf("      └─ Session:   %v timeout", cfg.Web.Auth.SessionTimeout))
		} else {
			lines = append(lines, "   └─ Auth:         Disabled")
		}
		exportStatus := "Disabled"
		if cfg.Web.Export.Enable {
			exportStatus = fmt.Sprintf("Enabled (%d format(s))", len(cfg.Web.Export.Formats))
		}
		lines = append(lines, fmt.Sprintf("   └─ Export:       %s", exportStatus))
	} else {
		lines = append(lines, "🖥️ Web Console:    Disabled")
	}

	// Forward target information
	lines = append(lines, "")
	if len(cfg.Forward.URLs) > 0 {
		lines = append(lines, fmt.Sprintf("🔀 Forward Targets:  %d Target(s)", len(cfg.Forward.URLs)))
		for _, url := range cfg.Forward.URLs {
			lines = append(lines, fmt.Sprintf("   └─ %s", url))
		}
	} else {
		lines = append(lines, "🔀 Forward Targets:  None")
	}

	// Mock responses summary
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("⚡ Mock Responses:  %s", formatMockResponseSummary(cfg)))

	// Path strategy summary
	lines = append(lines, fmt.Sprintf("🧩 Path Strategy:   %s", formatPathStrategySummary(cfg)))

	// Output mode summary
	lines = append(lines, fmt.Sprintf("🖨️ Output Mode:    %s (silence=%v)", strings.ToLower(cfg.Output.Mode), cfg.Output.Silence))
	if cfg.Output.BodyView.Enable {
		preview := "unlimited"
		if cfg.Output.BodyView.MaxPreviewBytes > 0 {
			preview = humanize.Bytes(uint64(cfg.Output.BodyView.MaxPreviewBytes))
		}
		lines = append(lines, fmt.Sprintf("🧾 Body View:      Enabled (preview=%s, full=%v)", preview, cfg.Output.BodyView.FullBody))
		lines = append(lines, fmt.Sprintf("   └─ JSON pretty: %v", cfg.Output.BodyView.Json.Pretty))
		if cfg.Output.BodyView.Binary.HexPreviewEnable {
			hexBytes := "auto"
			if cfg.Output.BodyView.Binary.HexPreviewBytes > 0 {
				hexBytes = humanize.Bytes(uint64(cfg.Output.BodyView.Binary.HexPreviewBytes))
			}
			lines = append(lines, fmt.Sprintf("   └─ Hex preview: %s", hexBytes))
		} else {
			lines = append(lines, "   └─ Hex preview: disabled")
		}
		if cfg.Output.BodyView.Binary.SaveToFile {
			dir := cfg.Output.BodyView.Binary.SaveDirectory
			if dir == "" {
				dir = "(cwd)"
			}
			lines = append(lines, fmt.Sprintf("   └─ Binary save: %s", dir))
		} else {
			lines = append(lines, "   └─ Binary save: disabled")
		}
	} else {
		lines = append(lines, "🧾 Body View:      Disabled")
	}

	// File logging information
	lines = append(lines, "")
	if cfg.Log.FileLogging.Enable {
		lines = append(lines, "💾 File Logging:    Enabled")
		compress := "Disabled"
		if cfg.Log.FileLogging.Compress {
			compress = "Enabled"
		}
		lines = append(lines, fmt.Sprintf("   └─ %s (%dMB, %d backups, %d days, compress: %s)",
			cfg.Log.FileLogging.Path,
			cfg.Log.FileLogging.MaxSizeMB,
			cfg.Log.FileLogging.MaxBackups,
			cfg.Log.FileLogging.MaxAgeDays,
			compress))
	} else {
		lines = append(lines, "💾 File Logging:    Disabled")
	}

	// Bottom information
	lines = append(lines, "")
	lines = append(lines, "(Press Ctrl+C to stop)")

	// Calculate maximum line length
	maxLength := 0
	for _, line := range lines {
		// Calculate display width (handling Chinese characters and emoji)
		lineLength := calculateDisplayWidth(line)
		if lineLength > maxLength {
			maxLength = lineLength
		}
	}

	// Set minimum width and margins
	boxWidth := maxLength + 4 // 2 characters margin on left and right
	if boxWidth < 50 {
		boxWidth = 50
	}

	// Print banner
	fmt.Println()
	printBoxTop(boxWidth)
	printBoxContent(titleLine, boxWidth, true)    // Center title
	printBoxContent(subtitleLine, boxWidth, true) // Center subtitle
	printBoxSeparator(boxWidth)

	// Print content lines (starting from line 4, skipping title and subtitle)
	for i := 3; i < len(lines); i++ {
		line := lines[i]
		printBoxContent(line, boxWidth, false)
	}

	printBoxBottom(boxWidth)
	fmt.Println()

	log.Info("ReqTap starting",
		"version", version,
		"port", cfg.Server.Port,
		"path", cfg.Server.Path,
		"log_level", cfg.Log.Level,
		"forward_urls", cfg.Forward.URLs,
		"web_enable", cfg.Web.Enable,
		"web_path", cfg.Web.Path,
		"web_admin_path", cfg.Web.AdminPath,
		"web_auth", cfg.Web.Auth.Enable,
		"web_export", cfg.Web.Export.Enable,
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// calculateDisplayWidth calculates the display width of a string, handling Chinese characters and emoji
func calculateDisplayWidth(s string) int {
	width := 0
	for _, r := range s {
		// Determine character width
		if r <= 127 {
			// ASCII character, width is 1
			width++
		} else if (r >= 0x1F600 && r <= 0x1F64F) || // Emoji emoticons
			(r >= 0x1F300 && r <= 0x1F5FF) || // Other symbols
			(r >= 0x1F680 && r <= 0x1F6FF) || // Transport and map symbols
			(r >= 0x2600 && r <= 0x26FF) || // Other symbols
			(r >= 0x2700 && r <= 0x27BF) { // Decorative symbols
			// Emoji, width is 2
			width += 2
		} else if (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
			(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
			(r >= 0x20000 && r <= 0x2A6DF) || // CJK Extension B
			(r >= 0x2A700 && r <= 0x2B73F) || // CJK Extension C
			(r >= 0x2B740 && r <= 0x2B81F) || // CJK Extension D
			(r >= 0x2B820 && r <= 0x2CEAF) || // CJK Extension E
			(r >= 0x2CEB0 && r <= 0x2EBEF) { // CJK Extension F
			// Chinese character, width is 2
			width += 2
		} else {
			// Other Unicode characters, usually width is 1
			width++
		}
	}
	return width
}

// printBoxTop prints the top border of the box
func printBoxTop(width int) {
	fmt.Printf("┌%s┐\n", strings.Repeat("─", width-2))
}

// printBoxBottom prints the bottom border of the box
func printBoxBottom(width int) {
	fmt.Printf("└%s┘\n", strings.Repeat("─", width-2))
}

// printBoxSeparator prints the separator line of the box
func printBoxSeparator(width int) {
	fmt.Printf("├%s┤\n", strings.Repeat("─", width-2))
}

func formatMockResponseSummary(cfg *config.Config) string {
	count := len(cfg.Server.Responses)
	if count == 0 {
		return "None configured"
	}
	var names []string
	for _, rule := range cfg.Server.Responses {
		names = append(names, rule.Name)
	}
	return fmt.Sprintf("%d rule(s): %s", count, strings.Join(names, ", "))
}

func formatPathStrategySummary(cfg *config.Config) string {
	mode := strings.ToLower(cfg.Forward.PathStrategy.Mode)
	if mode == "" {
		mode = "append"
	}
	switch mode {
	case "strip_prefix":
		prefix := cfg.Forward.PathStrategy.StripPrefix
		if prefix == "" {
			prefix = cfg.Server.Path
		}
		return fmt.Sprintf("strip_prefix (prefix=%s)", prefix)
	case "rewrite":
		ruleCount := len(cfg.Forward.PathStrategy.Rules)
		return fmt.Sprintf("rewrite (%d rule(s))", ruleCount)
	default:
		return "append"
	}
}

func logStartupSummary(cfg *config.Config, log logger.Logger) {
	mode := strings.ToLower(cfg.Output.Mode)
	var responseNames []string
	for _, rule := range cfg.Server.Responses {
		responseNames = append(responseNames, rule.Name)
	}
	log.Info("Startup configuration",
		"port", cfg.Server.Port,
		"path", cfg.Server.Path,
		"responses", responseNames,
		"forward_urls", cfg.Forward.URLs,
		"path_strategy", formatPathStrategySummary(cfg),
		"output_mode", mode,
		"silence", cfg.Output.Silence,
	)
}

// printBoxContent prints the content line of the box
func printBoxContent(content string, boxWidth int, center bool) {
	// Calculate the display width of the content
	contentWidth := calculateDisplayWidth(content)

	// Calculate required padding space
	padding := boxWidth - 2 - contentWidth // Subtract 1 character for left and right borders
	if padding < 0 {
		padding = 0
	}

	// Generate padding strings
	leftPad := ""
	rightPad := ""

	if center {
		// Center alignment
		leftPad = strings.Repeat(" ", padding/2)
		rightPad = strings.Repeat(" ", padding-padding/2)
	} else {
		// Left alignment
		leftPad = "  " // Fixed 2-space indentation on the left
		rightPad = strings.Repeat(" ", padding-2)
	}

	fmt.Printf("│%s%s%s│\n", leftPad, content, rightPad)
}

func validateWebPathConflicts(cfg *config.Config) error {
	if cfg == nil || !cfg.Web.Enable {
		return nil
	}

	serverPath := normalizeConfigPath(cfg.Server.Path)
	webPath := normalizeConfigPath(cfg.Web.Path)
	adminPath := normalizeConfigPath(cfg.Web.AdminPath)

	if pathsOverlap(serverPath, webPath) {
		return fmt.Errorf("web.path (%s) conflicts with server.path (%s); please configure different values", cfg.Web.Path, cfg.Server.Path)
	}
	if pathsOverlap(serverPath, adminPath) {
		return fmt.Errorf("web.admin_path (%s) conflicts with server.path (%s); please configure different values", cfg.Web.AdminPath, cfg.Server.Path)
	}

	return nil
}

func normalizeConfigPath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		p = strings.TrimRight(p, "/")
	}
	return p
}

func pathsOverlap(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == "/" && b == "/" {
		return true
	}
	if a == "/" || b == "/" {
		return false
	}
	if a == b {
		return true
	}

	aPrefix := strings.TrimRight(a, "/") + "/"
	bPrefix := strings.TrimRight(b, "/") + "/"

	return strings.HasPrefix(aPrefix, bPrefix) || strings.HasPrefix(bPrefix, aPrefix)
}
