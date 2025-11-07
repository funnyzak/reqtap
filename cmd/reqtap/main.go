package main

import (
	"fmt"
	"os"
	"strings"
	"time"

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

	// Web console configuration bindings
	viper.BindPFlag("web.enable", cmd.Flags().Lookup("web-enable"))
	viper.BindPFlag("web.path", cmd.Flags().Lookup("web-path"))
	viper.BindPFlag("web.admin_path", cmd.Flags().Lookup("web-admin-path"))
	viper.BindPFlag("web.max_requests", cmd.Flags().Lookup("web-max-requests"))
	viper.BindPFlag("web.auth.enable", cmd.Flags().Lookup("web-auth-enable"))
	viper.BindPFlag("web.auth.session_timeout", cmd.Flags().Lookup("web-auth-session-timeout"))
	viper.BindPFlag("web.export.enable", cmd.Flags().Lookup("web-export-enable"))
	viper.BindPFlag("web.export.formats", cmd.Flags().Lookup("web-export-formats"))
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

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	if err := validateWebPathConflicts(cfg); err != nil {
		return err
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
