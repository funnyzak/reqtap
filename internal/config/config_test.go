package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Test default configuration
	t.Run("Default config", func(t *testing.T) {
		cfg, err := LoadConfig("")
		if err != nil {
			t.Fatalf("Failed to load default config: %v", err)
		}

		// Verify default values
		if cfg.Server.Port != 38888 {
			t.Errorf("Expected default port 38888, got %d", cfg.Server.Port)
		}

		if cfg.Server.Path != "/" {
			t.Errorf("Expected default path '/', got %s", cfg.Server.Path)
		}

		if cfg.Log.Level != "info" {
			t.Errorf("Expected default log level 'info', got %s", cfg.Log.Level)
		}

		if cfg.Forward.Timeout != 30 {
			t.Errorf("Expected default forward timeout 30, got %d", cfg.Forward.Timeout)
		}

		if cfg.Forward.MaxRetries != 3 {
			t.Errorf("Expected default max retries 3, got %d", cfg.Forward.MaxRetries)
		}

		if cfg.Forward.MaxConcurrent != 10 {
			t.Errorf("Expected default max concurrent 10, got %d", cfg.Forward.MaxConcurrent)
		}
	})
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid config",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Path: "/webhook",
				},
				Log: LogConfig{
					Level: "info",
					FileLogging: FileLogConfig{
						Enable:     false,
						Path:       "./reqtap.log",
						MaxSizeMB:  10,
						MaxBackups: 5,
						MaxAgeDays: 30,
						Compress:   true,
					},
				},
				Forward: ForwardConfig{
					URLs:          []string{"http://localhost:3000"},
					Timeout:       30,
					MaxRetries:    3,
					MaxConcurrent: 10,
				},
			},
			expectError: false,
		},
		{
			name: "Invalid port",
			config: &Config{
				Server: ServerConfig{
					Port: 70000, // Out of range
					Path: "/",
				},
			},
			expectError: true,
			errorMsg:    "invalid port",
		},
		{
			name: "Empty path",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Path: "",
				},
			},
			expectError: true,
			errorMsg:    "server path cannot be empty",
		},
		{
			name: "Invalid log level",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Path: "/",
				},
				Log: LogConfig{
					Level: "invalid",
				},
			},
			expectError: true,
			errorMsg:    "invalid log level",
		},
		{
			name: "File logging enabled but empty path",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Path: "/",
				},
				Log: LogConfig{
					Level: "info",
					FileLogging: FileLogConfig{
						Enable: true,
						Path:   "", // Empty path
					},
				},
			},
			expectError: true,
			errorMsg:    "log file path cannot be empty",
		},
		{
			name: "Empty forward URL",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Path: "/",
				},
				Log: LogConfig{
					Level: "info", // Add valid log level
				},
				Forward: ForwardConfig{
					URLs: []string{"http://localhost:3000", ""}, // Contains empty URL
				},
			},
			expectError: true,
			errorMsg:    "forward URL 2 cannot be empty",
		},
		{
			name: "Negative forward timeout",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Path: "/",
				},
				Log: LogConfig{
					Level: "info", // Add valid log level
				},
				Forward: ForwardConfig{
					URLs:    []string{"http://localhost:3000"},
					Timeout: -1, // Negative number
				},
			},
			expectError: true,
			errorMsg:    "forward timeout cannot be negative",
		},
		{
			name: "Zero max concurrent",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Path: "/",
				},
				Log: LogConfig{
					Level: "info", // Add valid log level
				},
				Forward: ForwardConfig{
					URLs:          []string{"http://localhost:3000"},
					MaxConcurrent: 0, // Must be at least 1
				},
			},
			expectError: true,
			errorMsg:    "forward max concurrent must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got no error", tt.errorMsg)
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}
		})
	}
}

func TestLoadConfigWithFile(t *testing.T) {
	// Create temporary configuration file
	configContent := `
server:
  port: 9999
  path: "/test"

log:
  level: "debug"
  file_logging:
    enable: true
    path: "/tmp/test.log"
    max_size_mb: 5
    max_backups: 3
    max_age_days: 7
    compress: false

forward:
  urls:
    - "http://localhost:4000"
    - "https://api.example.com"
  timeout: 60
  max_retries: 5
  max_concurrent: 20
`

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "reqtap_test_config_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write configuration content
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config content: %v", err)
	}
	tmpFile.Close()

	// Load configuration
	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify configuration values
	if cfg.Server.Port != 9999 {
		t.Errorf("Expected port 9999, got %d", cfg.Server.Port)
	}

	if cfg.Server.Path != "/test" {
		t.Errorf("Expected path '/test', got %s", cfg.Server.Path)
	}

	if cfg.Log.Level != "debug" {
		t.Errorf("Expected log level 'debug', got %s", cfg.Log.Level)
	}

	if !cfg.Log.FileLogging.Enable {
		t.Error("Expected file logging to be enabled")
	}

	if cfg.Log.FileLogging.Path != "/tmp/test.log" {
		t.Errorf("Expected log path '/tmp/test.log', got %s", cfg.Log.FileLogging.Path)
	}

	if len(cfg.Forward.URLs) != 2 {
		t.Errorf("Expected 2 forward URLs, got %d", len(cfg.Forward.URLs))
	}

	if cfg.Forward.Timeout != 60 {
		t.Errorf("Expected forward timeout 60, got %d", cfg.Forward.Timeout)
	}
}

func TestLoadConfigInvalidFile(t *testing.T) {
	// Test that non-existent configuration file should return error
	cfg, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for missing config file")
	}
	if cfg != nil {
		t.Error("Expected nil config for missing file")
	}
}

// Helper function: check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	n := len(substr)
	if n == 0 {
		return 0
	}
	if n > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-n; i++ {
		if s[i:i+n] == substr {
			return i
		}
	}
	return -1
}
