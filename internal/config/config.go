package config

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config application configuration structure
type Config struct {
	Server  ServerConfig  `yaml:"server" mapstructure:"server"`
	Log     LogConfig     `yaml:"log" mapstructure:"log"`
	Forward ForwardConfig `yaml:"forward" mapstructure:"forward"`
	Web     WebConfig     `yaml:"web" mapstructure:"web"`
	Output  OutputConfig  `yaml:"output" mapstructure:"output"`
	Storage StorageConfig `yaml:"storage" mapstructure:"storage"`
}

// ServerConfig HTTP server configuration
type ServerConfig struct {
	Port int    `yaml:"port" mapstructure:"port"`
	Path string `yaml:"path" mapstructure:"path"`
	// MaxBodyBytes limits the size of accepted request bodies (0 = unlimited)
	MaxBodyBytes int64                     `yaml:"max_body_bytes" mapstructure:"max_body_bytes"`
	Responses    []ImmediateResponseConfig `yaml:"responses" mapstructure:"responses"`
}

// ImmediateResponseConfig describes an inline response rule for incoming requests
type ImmediateResponseConfig struct {
	Name       string            `yaml:"name" mapstructure:"name"`
	Methods    []string          `yaml:"methods" mapstructure:"methods"`
	Path       string            `yaml:"path" mapstructure:"path"`
	PathPrefix string            `yaml:"path_prefix" mapstructure:"path_prefix"`
	Status     int               `yaml:"status" mapstructure:"status"`
	Body       string            `yaml:"body" mapstructure:"body"`
	Headers    map[string]string `yaml:"headers" mapstructure:"headers"`
}

// LogConfig log configuration
type LogConfig struct {
	Level       string        `yaml:"level"`
	FileLogging FileLogConfig `yaml:"file_logging"`
}

// FileLogConfig file log configuration
type FileLogConfig struct {
	Enable     bool   `yaml:"enable"`
	Path       string `yaml:"path"`
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAgeDays int    `yaml:"max_age_days"`
	Compress   bool   `yaml:"compress"`
}

// ForwardConfig forwarding configuration
type ForwardConfig struct {
	URLs                  []string                  `yaml:"urls" mapstructure:"urls"`
	Timeout               int                       `yaml:"timeout" mapstructure:"timeout"`
	MaxRetries            int                       `yaml:"max_retries" mapstructure:"max_retries"`
	MaxConcurrent         int                       `yaml:"max_concurrent" mapstructure:"max_concurrent"`
	MaxIdleConns          int                       `yaml:"max_idle_conns" mapstructure:"max_idle_conns"`
	MaxIdleConnsPerHost   int                       `yaml:"max_idle_conns_per_host" mapstructure:"max_idle_conns_per_host"`
	MaxConnsPerHost       int                       `yaml:"max_conns_per_host" mapstructure:"max_conns_per_host"`
	IdleConnTimeout       int                       `yaml:"idle_conn_timeout" mapstructure:"idle_conn_timeout"`
	ResponseHeaderTimeout int                       `yaml:"response_header_timeout" mapstructure:"response_header_timeout"`
	TLSHandshakeTimeout   int                       `yaml:"tls_handshake_timeout" mapstructure:"tls_handshake_timeout"`
	ExpectContinueTimeout int                       `yaml:"expect_continue_timeout" mapstructure:"expect_continue_timeout"`
	TLSInsecureSkipVerify bool                      `yaml:"tls_insecure_skip_verify" mapstructure:"tls_insecure_skip_verify"`
	PathStrategy          ForwardPathStrategyConfig `yaml:"path_strategy" mapstructure:"path_strategy"`
	HeaderBlacklist       []string                  `yaml:"header_blacklist" mapstructure:"header_blacklist"`
	HeaderWhitelist       []string                  `yaml:"header_whitelist" mapstructure:"header_whitelist"`
}

// ForwardPathStrategyConfig configures how target paths are constructed
type ForwardPathStrategyConfig struct {
	Mode        string                     `yaml:"mode" mapstructure:"mode"`
	StripPrefix string                     `yaml:"strip_prefix" mapstructure:"strip_prefix"`
	Rules       []ForwardRewriteRuleConfig `yaml:"rules" mapstructure:"rules"`
}

// ForwardRewriteRuleConfig defines a rewrite rule when mode is rewrite
type ForwardRewriteRuleConfig struct {
	Name    string `yaml:"name" mapstructure:"name"`
	Match   string `yaml:"match" mapstructure:"match"`
	Replace string `yaml:"replace" mapstructure:"replace"`
	Regex   bool   `yaml:"regex" mapstructure:"regex"`
}

// WebConfig web console configuration
type WebConfig struct {
	Enable           bool            `yaml:"enable" mapstructure:"enable"`
	Path             string          `yaml:"path" mapstructure:"path"`
	AdminPath        string          `yaml:"admin_path" mapstructure:"admin_path"`
	MaxRequests      int             `yaml:"max_requests" mapstructure:"max_requests"`
	DefaultLocale    string          `yaml:"default_locale" mapstructure:"default_locale"`
	SupportedLocales []string        `yaml:"supported_locales" mapstructure:"supported_locales"`
	Auth             WebAuthConfig   `yaml:"auth" mapstructure:"auth"`
	Export           WebExportConfig `yaml:"export" mapstructure:"export"`
}

// WebAuthConfig authentication configuration
type WebAuthConfig struct {
	Enable         bool            `yaml:"enable" mapstructure:"enable"`
	SessionTimeout time.Duration   `yaml:"session_timeout" mapstructure:"session_timeout"`
	Users          []WebUserConfig `yaml:"users" mapstructure:"users"`
}

// WebUserConfig user credential configuration
type WebUserConfig struct {
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
	Role     string `yaml:"role" mapstructure:"role"`
}

// WebExportConfig export configuration
type WebExportConfig struct {
	Enable  bool     `yaml:"enable" mapstructure:"enable"`
	Formats []string `yaml:"formats" mapstructure:"formats"`
}

// OutputConfig controls CLI output style
type OutputConfig struct {
	Mode     string         `yaml:"mode" mapstructure:"mode"`
	Silence  bool           `yaml:"silence" mapstructure:"silence"`
	Locale   string         `yaml:"locale" mapstructure:"locale"`
	BodyView BodyViewConfig `yaml:"body_view" mapstructure:"body_view"`
}

// StorageConfig 持久化存储参数
type StorageConfig struct {
	Driver     string        `yaml:"driver" mapstructure:"driver"`
	Path       string        `yaml:"path" mapstructure:"path"`
	MaxRecords int           `yaml:"max_records" mapstructure:"max_records"`
	Retention  time.Duration `yaml:"retention" mapstructure:"retention"`
}

// BodyViewConfig 控制正文格式化与分段
type BodyViewConfig struct {
	Enable          bool             `yaml:"enable" mapstructure:"enable"`
	MaxPreviewBytes int              `yaml:"max_preview_bytes" mapstructure:"max_preview_bytes"`
	FullBody        bool             `yaml:"full_body" mapstructure:"full_body"`
	Json            JSONViewConfig   `yaml:"json" mapstructure:"json"`
	Form            FormViewConfig   `yaml:"form" mapstructure:"form"`
	XML             XMLViewConfig    `yaml:"xml" mapstructure:"xml"`
	HTML            HTMLViewConfig   `yaml:"html" mapstructure:"html"`
	Binary          BinaryViewConfig `yaml:"binary" mapstructure:"binary"`
}

// JSONViewConfig JSON 展示参数
type JSONViewConfig struct {
	Enable         bool `yaml:"enable" mapstructure:"enable"`
	Pretty         bool `yaml:"pretty" mapstructure:"pretty"`
	MaxIndentBytes int  `yaml:"max_indent_bytes" mapstructure:"max_indent_bytes"`
}

// FormViewConfig 表单展示参数
type FormViewConfig struct {
	Enable bool `yaml:"enable" mapstructure:"enable"`
}

// XMLViewConfig XML 展示参数
type XMLViewConfig struct {
	Enable       bool `yaml:"enable" mapstructure:"enable"`
	Pretty       bool `yaml:"pretty" mapstructure:"pretty"`
	StripControl bool `yaml:"strip_control" mapstructure:"strip_control"`
}

// HTMLViewConfig HTML 展示参数
type HTMLViewConfig struct {
	Enable       bool `yaml:"enable" mapstructure:"enable"`
	Pretty       bool `yaml:"pretty" mapstructure:"pretty"`
	StripControl bool `yaml:"strip_control" mapstructure:"strip_control"`
}

// BinaryViewConfig 二进制展示参数
type BinaryViewConfig struct {
	HexPreviewEnable bool   `yaml:"hex_preview_enable" mapstructure:"hex_preview_enable"`
	HexPreviewBytes  int    `yaml:"hex_preview_bytes" mapstructure:"hex_preview_bytes"`
	SaveToFile       bool   `yaml:"save_to_file" mapstructure:"save_to_file"`
	SaveDirectory    string `yaml:"save_directory" mapstructure:"save_directory"`
}

// LoadConfig load configuration
// If v is nil, a new viper instance will be created
func LoadConfig(configPath string, v *viper.Viper) (*Config, error) {
	if v == nil {
		v = viper.New()
	}

	// Set default values
	setDefaults(v)

	// Set environment variable prefix
	v.SetEnvPrefix("REQTAP")
	v.AutomaticEnv()

	// Set configuration file
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Configuration file search paths
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.reqtap")
		v.AddConfigPath("/etc/reqtap")
	}

	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("No config file found, using defaults")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	} else {
		log.Printf("Config file loaded: %s", v.ConfigFileUsed())
	}

	// Continue parsing default values even without configuration file

	// Unmarshal to struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	// Ensure zero-value fields use default values (Unmarshal doesn't apply defaults to zero-value fields)
	// Also override with command line flags if they are set (command line flags have highest priority)
	applyDefaults(&config, v)

	return &config, nil
}

// applyDefaults apply default values to zero-value fields in the struct
// This only applies defaults for fields that don't have command line flags.
// Command line flags are handled separately in main.go to ensure highest priority.
func applyDefaults(cfg *Config, v *viper.Viper) {
	// Server configuration - only apply defaults if zero (command line handled in main.go)
	if cfg.Server.Port == 0 {
		cfg.Server.Port = v.GetInt("server.port")
	}
	if cfg.Server.Path == "" {
		cfg.Server.Path = v.GetString("server.path")
	}
	if cfg.Server.MaxBodyBytes == 0 {
		cfg.Server.MaxBodyBytes = v.GetInt64("server.max_body_bytes")
	}
	if len(cfg.Server.Responses) == 0 {
		var defaults []ImmediateResponseConfig
		if err := v.UnmarshalKey("server.responses", &defaults); err == nil {
			cfg.Server.Responses = defaults
		}
	}
	for i := range cfg.Server.Responses {
		cfg.Server.Responses[i].Headers = canonicalizeHeaders(cfg.Server.Responses[i].Headers)
	}

	// Log configuration - only apply defaults if zero (command line handled in main.go)
	if cfg.Log.Level == "" {
		cfg.Log.Level = v.GetString("log.level")
	}

	// File logging configuration - only apply defaults if zero (command line handled in main.go)
	// Note: For bool fields, we always use viper's value since it correctly handles
	// both config file values and defaults. Viper.GetBool() will return the value from
	// config file if set, otherwise the default value.
	cfg.Log.FileLogging.Enable = v.GetBool("log.file_logging.enable")
	cfg.Log.FileLogging.Compress = v.GetBool("log.file_logging.compress")
	if cfg.Log.FileLogging.Path == "" {
		cfg.Log.FileLogging.Path = v.GetString("log.file_logging.path")
	}
	if cfg.Log.FileLogging.MaxSizeMB == 0 {
		cfg.Log.FileLogging.MaxSizeMB = v.GetInt("log.file_logging.max_size_mb")
	}
	if cfg.Log.FileLogging.MaxBackups == 0 {
		cfg.Log.FileLogging.MaxBackups = v.GetInt("log.file_logging.max_backups")
	}
	if cfg.Log.FileLogging.MaxAgeDays == 0 {
		cfg.Log.FileLogging.MaxAgeDays = v.GetInt("log.file_logging.max_age_days")
	}

	// Output configuration
	if cfg.Output.Mode == "" {
		cfg.Output.Mode = v.GetString("output.mode")
	}
	cfg.Output.Silence = v.GetBool("output.silence")
	cfg.Output.BodyView.Enable = v.GetBool("output.body_view.enable")
	if cfg.Output.BodyView.MaxPreviewBytes == 0 {
		cfg.Output.BodyView.MaxPreviewBytes = v.GetInt("output.body_view.max_preview_bytes")
	}
	cfg.Output.BodyView.FullBody = v.GetBool("output.body_view.full_body")
	cfg.Output.BodyView.Json.Enable = v.GetBool("output.body_view.json.enable")
	cfg.Output.BodyView.Json.Pretty = v.GetBool("output.body_view.json.pretty")
	if cfg.Output.BodyView.Json.MaxIndentBytes == 0 {
		cfg.Output.BodyView.Json.MaxIndentBytes = v.GetInt("output.body_view.json.max_indent_bytes")
	}
	cfg.Output.BodyView.Form.Enable = v.GetBool("output.body_view.form.enable")
	cfg.Output.BodyView.XML.Enable = v.GetBool("output.body_view.xml.enable")
	cfg.Output.BodyView.XML.Pretty = v.GetBool("output.body_view.xml.pretty")
	cfg.Output.BodyView.XML.StripControl = v.GetBool("output.body_view.xml.strip_control")
	cfg.Output.BodyView.HTML.Enable = v.GetBool("output.body_view.html.enable")
	cfg.Output.BodyView.HTML.Pretty = v.GetBool("output.body_view.html.pretty")
	cfg.Output.BodyView.HTML.StripControl = v.GetBool("output.body_view.html.strip_control")
	cfg.Output.BodyView.Binary.HexPreviewEnable = v.GetBool("output.body_view.binary.hex_preview_enable")
	if cfg.Output.BodyView.Binary.HexPreviewBytes == 0 {
		cfg.Output.BodyView.Binary.HexPreviewBytes = v.GetInt("output.body_view.binary.hex_preview_bytes")
	}
	cfg.Output.BodyView.Binary.SaveToFile = v.GetBool("output.body_view.binary.save_to_file")
	if cfg.Output.BodyView.Binary.SaveDirectory == "" {
		cfg.Output.BodyView.Binary.SaveDirectory = v.GetString("output.body_view.binary.save_directory")
	}

	// Forward configuration - command line handled in main.go for URLs
	// These don't have command line flags, so only apply defaults if zero
	if cfg.Forward.Timeout == 0 {
		cfg.Forward.Timeout = v.GetInt("forward.timeout")
	}
	if cfg.Forward.MaxRetries == 0 {
		cfg.Forward.MaxRetries = v.GetInt("forward.max_retries")
	}
	// MaxConcurrent must be at least 1, so use default value if 0
	if cfg.Forward.MaxConcurrent == 0 {
		cfg.Forward.MaxConcurrent = v.GetInt("forward.max_concurrent")
	}
	if cfg.Forward.MaxIdleConns == 0 {
		cfg.Forward.MaxIdleConns = v.GetInt("forward.max_idle_conns")
	}
	if cfg.Forward.MaxIdleConnsPerHost == 0 {
		cfg.Forward.MaxIdleConnsPerHost = v.GetInt("forward.max_idle_conns_per_host")
	}
	if cfg.Forward.MaxConnsPerHost == 0 {
		cfg.Forward.MaxConnsPerHost = v.GetInt("forward.max_conns_per_host")
	}
	if cfg.Forward.IdleConnTimeout == 0 {
		cfg.Forward.IdleConnTimeout = v.GetInt("forward.idle_conn_timeout")
	}
	if cfg.Forward.ResponseHeaderTimeout == 0 {
		cfg.Forward.ResponseHeaderTimeout = v.GetInt("forward.response_header_timeout")
	}
	if cfg.Forward.TLSHandshakeTimeout == 0 {
		cfg.Forward.TLSHandshakeTimeout = v.GetInt("forward.tls_handshake_timeout")
	}
	if cfg.Forward.ExpectContinueTimeout == 0 {
		cfg.Forward.ExpectContinueTimeout = v.GetInt("forward.expect_continue_timeout")
	}
	if cfg.Forward.PathStrategy.Mode == "" {
		cfg.Forward.PathStrategy.Mode = v.GetString("forward.path_strategy.mode")
	}
	if cfg.Forward.PathStrategy.StripPrefix == "" {
		cfg.Forward.PathStrategy.StripPrefix = v.GetString("forward.path_strategy.strip_prefix")
	}
	if len(cfg.Forward.PathStrategy.Rules) == 0 {
		var rules []ForwardRewriteRuleConfig
		if err := v.UnmarshalKey("forward.path_strategy.rules", &rules); err == nil {
			cfg.Forward.PathStrategy.Rules = rules
		}
	}
	if len(cfg.Forward.HeaderBlacklist) == 0 {
		cfg.Forward.HeaderBlacklist = v.GetStringSlice("forward.header_blacklist")
	}
	if len(cfg.Forward.HeaderWhitelist) == 0 {
		cfg.Forward.HeaderWhitelist = v.GetStringSlice("forward.header_whitelist")
	}
	cfg.Forward.HeaderBlacklist = normalizeHeaderList(cfg.Forward.HeaderBlacklist)
	cfg.Forward.HeaderWhitelist = normalizeHeaderList(cfg.Forward.HeaderWhitelist)
	cfg.Forward.TLSInsecureSkipVerify = v.GetBool("forward.tls_insecure_skip_verify")

	// Web configuration defaults
	cfg.Web.Enable = v.GetBool("web.enable")
	if cfg.Web.Path == "" {
		cfg.Web.Path = v.GetString("web.path")
	}
	if cfg.Web.AdminPath == "" {
		cfg.Web.AdminPath = v.GetString("web.admin_path")
	}
	if cfg.Web.MaxRequests == 0 {
		cfg.Web.MaxRequests = v.GetInt("web.max_requests")
	}

	// Auth defaults
	cfg.Web.Auth.Enable = v.GetBool("web.auth.enable")
	if cfg.Web.Auth.SessionTimeout == 0 {
		timeoutStr := v.GetString("web.auth.session_timeout")
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			cfg.Web.Auth.SessionTimeout = timeout
		} else {
			cfg.Web.Auth.SessionTimeout = 24 * time.Hour
		}
	}
	if len(cfg.Web.Auth.Users) == 0 {
		var users []WebUserConfig
		if err := v.UnmarshalKey("web.auth.users", &users); err == nil {
			cfg.Web.Auth.Users = users
		}
	}

	// Export defaults
	cfg.Web.Export.Enable = v.GetBool("web.export.enable")
	if len(cfg.Web.Export.Formats) == 0 {
		cfg.Web.Export.Formats = v.GetStringSlice("web.export.formats")
	}
}

// setDefaults set default configuration values
func setDefaults(v *viper.Viper) {
	// Server default configuration
	v.SetDefault("server.port", 38888)
	v.SetDefault("server.path", "/reqtap")
	v.SetDefault("server.max_body_bytes", int64(10*1024*1024))
	v.SetDefault("server.responses", []map[string]interface{}{
		{
			"name":   "default-ok",
			"status": 200,
			"body":   "ok",
			"headers": map[string]string{
				"Content-Type": "text/plain",
			},
		},
	})

	// Log default configuration
	v.SetDefault("log.level", "info")
	v.SetDefault("log.file_logging.enable", false)
	v.SetDefault("log.file_logging.path", "./reqtap.log")
	v.SetDefault("log.file_logging.max_size_mb", 10)
	v.SetDefault("log.file_logging.max_backups", 5)
	v.SetDefault("log.file_logging.max_age_days", 30)
	v.SetDefault("log.file_logging.compress", true)

	// Forward default configuration
	v.SetDefault("forward.urls", []string{})
	v.SetDefault("forward.timeout", 30)
	v.SetDefault("forward.max_retries", 3)
	v.SetDefault("forward.max_concurrent", 10)
	v.SetDefault("forward.max_idle_conns", 200)
	v.SetDefault("forward.max_idle_conns_per_host", 50)
	v.SetDefault("forward.max_conns_per_host", 100)
	v.SetDefault("forward.idle_conn_timeout", 90)
	v.SetDefault("forward.response_header_timeout", 15)
	v.SetDefault("forward.tls_handshake_timeout", 10)
	v.SetDefault("forward.expect_continue_timeout", 1)
	v.SetDefault("forward.tls_insecure_skip_verify", false)
	v.SetDefault("forward.path_strategy.mode", "append")
	v.SetDefault("forward.path_strategy.strip_prefix", "")
	v.SetDefault("forward.path_strategy.rules", []map[string]string{})
	v.SetDefault("forward.header_blacklist", []string{
		"host",
		"connection",
		"keep-alive",
		"proxy-authenticate",
		"proxy-authorization",
		"te",
		"trailers",
		"transfer-encoding",
		"upgrade",
		"content-length",
	})
	v.SetDefault("forward.header_whitelist", []string{})

	// Web console defaults
	v.SetDefault("web.enable", true)
	v.SetDefault("web.path", "/web")
	v.SetDefault("web.admin_path", "/api")
	v.SetDefault("web.max_requests", 500)
	v.SetDefault("web.default_locale", "en")
	v.SetDefault("web.supported_locales", []string{"en", "zh-CN", "ja", "ko", "fr", "ru"})
	v.SetDefault("web.auth.enable", true)
	v.SetDefault("web.auth.session_timeout", "24h")
	v.SetDefault("web.auth.users", []map[string]string{
		{"username": "admin", "password": "admin123", "role": "admin"},
		{"username": "user", "password": "user123", "role": "viewer"},
	})
	v.SetDefault("web.export.enable", true)
	v.SetDefault("web.export.formats", []string{"json", "csv", "txt"})

	// Output defaults
	v.SetDefault("output.mode", "console")
	v.SetDefault("output.silence", false)
	v.SetDefault("output.locale", "en")
	v.SetDefault("output.body_view.enable", false)
	v.SetDefault("output.body_view.max_preview_bytes", int(32*1024))
	v.SetDefault("output.body_view.full_body", false)
	v.SetDefault("output.body_view.json.enable", true)
	v.SetDefault("output.body_view.json.pretty", true)
	v.SetDefault("output.body_view.json.max_indent_bytes", int(128*1024))
	v.SetDefault("output.body_view.form.enable", true)
	v.SetDefault("output.body_view.xml.enable", true)
	v.SetDefault("output.body_view.xml.pretty", true)
	v.SetDefault("output.body_view.xml.strip_control", true)
	v.SetDefault("output.body_view.html.enable", true)
	v.SetDefault("output.body_view.html.pretty", false)
	v.SetDefault("output.body_view.html.strip_control", true)
	v.SetDefault("output.body_view.binary.hex_preview_enable", false)
	v.SetDefault("output.body_view.binary.hex_preview_bytes", 256)
	v.SetDefault("output.body_view.binary.save_to_file", false)
	v.SetDefault("output.body_view.binary.save_directory", "")

	// Storage defaults
	v.SetDefault("storage.driver", "sqlite")
	v.SetDefault("storage.path", "./data/reqtap.db")
	v.SetDefault("storage.max_records", 100000)
	v.SetDefault("storage.retention", "0s")
}

// validate configuration
func (c *Config) Validate() error {
	// Validate port
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", c.Server.Port)
	}

	// Validate path
	if c.Server.Path == "" {
		return fmt.Errorf("server path cannot be empty")
	}
	if c.Server.MaxBodyBytes < 0 {
		return fmt.Errorf("server max body bytes cannot be negative")
	}
	if len(c.Server.Responses) == 0 {
		return fmt.Errorf("server responses configuration cannot be empty")
	}
	for i, resp := range c.Server.Responses {
		if resp.Status < 100 || resp.Status > 599 {
			return fmt.Errorf("server response %d status must be between 100 and 599", i+1)
		}
		if resp.Path != "" && !strings.HasPrefix(resp.Path, "/") {
			return fmt.Errorf("server response %d path must start with '/'", i+1)
		}
		if resp.PathPrefix != "" && !strings.HasPrefix(resp.PathPrefix, "/") {
			return fmt.Errorf("server response %d path_prefix must start with '/'", i+1)
		}
		for _, method := range resp.Methods {
			if method == "" {
				return fmt.Errorf("server response %d contains empty method", i+1)
			}
		}
	}

	switch strings.ToLower(c.Output.Mode) {
	case "", "console", "json":
		if c.Output.Mode == "" {
			c.Output.Mode = "console"
		}
	default:
		return fmt.Errorf("output mode must be 'console' or 'json'")
	}
	if err := validateBodyViewConfig(&c.Output.BodyView); err != nil {
		return err
	}

	switch strings.ToLower(strings.TrimSpace(c.Storage.Driver)) {
	case "", "sqlite", "sqlite3":
		if strings.TrimSpace(c.Storage.Driver) == "" {
			c.Storage.Driver = "sqlite"
		}
	default:
		return fmt.Errorf("storage driver must be sqlite")
	}
	if strings.TrimSpace(c.Storage.Path) == "" {
		return fmt.Errorf("storage path cannot be empty")
	}
	if c.Storage.MaxRecords < 0 {
		return fmt.Errorf("storage max_records cannot be negative")
	}
	if c.Storage.Retention < 0 {
		return fmt.Errorf("storage retention cannot be negative")
	}

	if strings.TrimSpace(c.Output.Locale) == "" {
		c.Output.Locale = "en"
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"trace": true, "debug": true, "info": true,
		"warn": true, "error": true, "fatal": true, "panic": true,
	}
	if !validLogLevels[c.Log.Level] {
		return fmt.Errorf("invalid log level: %s", c.Log.Level)
	}

	// Validate file log configuration
	if c.Log.FileLogging.Enable {
		if c.Log.FileLogging.Path == "" {
			return fmt.Errorf("log file path cannot be empty when file logging is enabled")
		}
		if c.Log.FileLogging.MaxSizeMB < 1 {
			return fmt.Errorf("log file max size must be at least 1MB")
		}
		if c.Log.FileLogging.MaxBackups < 0 {
			return fmt.Errorf("log file max backups cannot be negative")
		}
		if c.Log.FileLogging.MaxAgeDays < 0 {
			return fmt.Errorf("log file max age cannot be negative")
		}
	}

	// Validate forward URLs
	for i, url := range c.Forward.URLs {
		if url == "" {
			return fmt.Errorf("forward URL %d cannot be empty", i+1)
		}
	}

	// Validate forward configuration
	if c.Forward.Timeout < 0 {
		return fmt.Errorf("forward timeout cannot be negative")
	}
	if c.Forward.MaxRetries < 0 {
		return fmt.Errorf("forward max retries cannot be negative")
	}
	if c.Forward.MaxConcurrent < 1 {
		return fmt.Errorf("forward max concurrent must be at least 1")
	}
	switch strings.ToLower(c.Forward.PathStrategy.Mode) {
	case "", "append", "strip_prefix", "rewrite":
		if c.Forward.PathStrategy.Mode == "" {
			c.Forward.PathStrategy.Mode = "append"
		}
	default:
		return fmt.Errorf("forward path strategy mode must be append, strip_prefix, or rewrite")
	}
	if strings.ToLower(c.Forward.PathStrategy.Mode) == "rewrite" {
		if len(c.Forward.PathStrategy.Rules) == 0 {
			return fmt.Errorf("forward path strategy rules cannot be empty when mode is rewrite")
		}
		for i, rule := range c.Forward.PathStrategy.Rules {
			if rule.Match == "" {
				return fmt.Errorf("forward path rule %d match cannot be empty", i+1)
			}
		}
	}

	for i, h := range c.Forward.HeaderBlacklist {
		if strings.TrimSpace(h) == "" {
			return fmt.Errorf("forward header_blacklist[%d] cannot be empty", i)
		}
	}
	for i, h := range c.Forward.HeaderWhitelist {
		if strings.TrimSpace(h) == "" {
			return fmt.Errorf("forward header_whitelist[%d] cannot be empty", i)
		}
	}

	// Validate web configuration
	if c.Web.Enable {
		if c.Web.Path == "" {
			return fmt.Errorf("web path cannot be empty")
		}
		if !strings.HasPrefix(c.Web.Path, "/") {
			return fmt.Errorf("web path must start with '/'")
		}
		if c.Web.AdminPath == "" {
			return fmt.Errorf("web admin path cannot be empty")
		}
		if !strings.HasPrefix(c.Web.AdminPath, "/") {
			return fmt.Errorf("web admin path must start with '/'")
		}
		if c.Web.MaxRequests < 1 {
			return fmt.Errorf("web max requests must be at least 1")
		}

		if c.Web.Auth.Enable {
			if c.Web.Auth.SessionTimeout <= 0 {
				return fmt.Errorf("web auth session timeout must be greater than zero")
			}
			if len(c.Web.Auth.Users) == 0 {
				return fmt.Errorf("web auth requires at least one user")
			}
			validRoles := map[string]struct{}{"admin": {}, "viewer": {}}
			for i, user := range c.Web.Auth.Users {
				if user.Username == "" {
					return fmt.Errorf("web auth user %d username cannot be empty", i+1)
				}
				if user.Password == "" {
					return fmt.Errorf("web auth user %d password cannot be empty", i+1)
				}
				if user.Role == "" {
					return fmt.Errorf("web auth user %d role cannot be empty", i+1)
				}
				if _, ok := validRoles[strings.ToLower(user.Role)]; !ok {
					return fmt.Errorf("web auth user %d role must be admin or viewer", i+1)
				}
			}
		}

		if c.Web.Export.Enable {
			if len(c.Web.Export.Formats) == 0 {
				return fmt.Errorf("web export formats cannot be empty when export enabled")
			}
		}
	}

	if strings.TrimSpace(c.Web.DefaultLocale) == "" {
		c.Web.DefaultLocale = "en"
	}
	c.Web.SupportedLocales = normalizeLocaleList(c.Web.SupportedLocales)
	if len(c.Web.SupportedLocales) == 0 {
		c.Web.SupportedLocales = []string{"en"}
	}
	if !containsLocale(c.Web.SupportedLocales, c.Web.DefaultLocale) {
		c.Web.SupportedLocales = append(c.Web.SupportedLocales, c.Web.DefaultLocale)
	}

	return nil
}

func validateBodyViewConfig(cfg *BodyViewConfig) error {
	if cfg.MaxPreviewBytes < 0 {
		return fmt.Errorf("output.body_view.max_preview_bytes cannot be negative")
	}
	if cfg.Json.MaxIndentBytes < 0 {
		return fmt.Errorf("output.body_view.json.max_indent_bytes cannot be negative")
	}
	if cfg.Binary.HexPreviewBytes < 0 {
		return fmt.Errorf("output.body_view.binary.hex_preview_bytes cannot be negative")
	}
	if cfg.Binary.SaveToFile && strings.TrimSpace(cfg.Binary.SaveDirectory) == "" {
		return fmt.Errorf("output.body_view.binary.save_directory cannot be empty when save_to_file is enabled")
	}
	return nil
}

func canonicalizeHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return headers
	}
	canonical := make(map[string]string, len(headers))
	for key, value := range headers {
		canonical[http.CanonicalHeaderKey(key)] = value
	}
	return canonical
}

func normalizeHeaderList(list []string) []string {
	if len(list) == 0 {
		return list
	}
	set := make(map[string]struct{}, len(list))
	result := make([]string, 0, len(list))
	for _, h := range list {
		norm := strings.ToLower(strings.TrimSpace(h))
		if norm == "" {
			continue
		}
		if _, exists := set[norm]; exists {
			continue
		}
		set[norm] = struct{}{}
		result = append(result, norm)
	}
	return result
}

func normalizeLocaleList(locales []string) []string {
	if len(locales) == 0 {
		return locales
	}
	set := make(map[string]struct{}, len(locales))
	result := make([]string, 0, len(locales))
	for _, loc := range locales {
		norm := strings.TrimSpace(loc)
		if norm == "" {
			continue
		}
		key := strings.ToLower(norm)
		if _, exists := set[key]; exists {
			continue
		}
		set[key] = struct{}{}
		result = append(result, norm)
	}
	return result
}

func containsLocale(locales []string, target string) bool {
	targetKey := strings.ToLower(strings.TrimSpace(target))
	for _, loc := range locales {
		if strings.ToLower(strings.TrimSpace(loc)) == targetKey {
			return true
		}
	}
	return false
}
