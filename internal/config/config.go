package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config application configuration structure
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Log     LogConfig     `yaml:"log"`
	Forward ForwardConfig `yaml:"forward"`
	Web     WebConfig     `yaml:"web"`
}

// ServerConfig HTTP server configuration
type ServerConfig struct {
	Port int    `yaml:"port"`
	Path string `yaml:"path"`
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
	URLs          []string `yaml:"urls"`
	Timeout       int      `yaml:"timeout"`
	MaxRetries    int      `yaml:"max_retries"`
	MaxConcurrent int      `yaml:"max_concurrent"`
}

// WebConfig web console configuration
type WebConfig struct {
	Enable      bool            `yaml:"enable"`
	Path        string          `yaml:"path"`
	AdminPath   string          `yaml:"admin_path"`
	MaxRequests int             `yaml:"max_requests"`
	Auth        WebAuthConfig   `yaml:"auth"`
	Export      WebExportConfig `yaml:"export"`
}

// WebAuthConfig authentication configuration
type WebAuthConfig struct {
	Enable         bool            `yaml:"enable"`
	SessionTimeout time.Duration   `yaml:"session_timeout"`
	Users          []WebUserConfig `yaml:"users"`
}

// WebUserConfig user credential configuration
type WebUserConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Role     string `yaml:"role"`
}

// WebExportConfig export configuration
type WebExportConfig struct {
	Enable  bool     `yaml:"enable"`
	Formats []string `yaml:"formats"`
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
	v.SetDefault("server.path", "/")

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

	// Web console defaults
	v.SetDefault("web.enable", true)
	v.SetDefault("web.path", "/web")
	v.SetDefault("web.admin_path", "/api")
	v.SetDefault("web.max_requests", 500)
	v.SetDefault("web.auth.enable", true)
	v.SetDefault("web.auth.session_timeout", "24h")
	v.SetDefault("web.auth.users", []map[string]string{
		{"username": "admin", "password": generateRandomPassword(10), "role": "admin"},
		{"username": "user", "password": generateRandomPassword(10), "role": "viewer"},
	})
	v.SetDefault("web.export.enable", true)
	v.SetDefault("web.export.formats", []string{"json", "csv", "txt"})
}

func generateRandomPassword(length int) string {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(buf)
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

	return nil
}
