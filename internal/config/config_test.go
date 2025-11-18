package config

import (
	"os"
	"testing"
	"time"
)

func defaultResponses() []ImmediateResponseConfig {
	return []ImmediateResponseConfig{
		{
			Name:    "default",
			Status:  200,
			Body:    "ok",
			Headers: map[string]string{"Content-Type": "text/plain"},
		},
	}
}

func TestLoadConfig(t *testing.T) {
	// Test default configuration
	t.Run("Default config", func(t *testing.T) {
		cfg, err := LoadConfig("", nil)
		if err != nil {
			t.Fatalf("Failed to load default config: %v", err)
		}

		// Verify default values
		if cfg.Server.Port != 38888 {
			t.Errorf("Expected default port 38888, got %d", cfg.Server.Port)
		}

		if cfg.Server.Path != "/reqtap" {
			t.Errorf("Expected default path '/reqtap', got %s", cfg.Server.Path)
		}

		if len(cfg.Server.Responses) == 0 {
			t.Fatalf("Expected default immediate responses to be populated")
		}

		if cfg.Server.Responses[0].Status != 200 {
			t.Errorf("Expected default response status 200, got %d", cfg.Server.Responses[0].Status)
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

		if cfg.Forward.PathStrategy.Mode != "append" {
			t.Errorf("Expected default path strategy 'append', got %s", cfg.Forward.PathStrategy.Mode)
		}

		if !cfg.Web.Enable {
			t.Errorf("Expected web console enabled by default")
		}

		if cfg.Web.Path != "/web" {
			t.Errorf("Expected default web path '/web', got %s", cfg.Web.Path)
		}

		if cfg.Web.AdminPath != "/api" {
			t.Errorf("Expected default web admin path '/api', got %s", cfg.Web.AdminPath)
		}

		if cfg.Web.MaxRequests != 500 {
			t.Errorf("Expected default max requests 500, got %d", cfg.Web.MaxRequests)
		}

		if !cfg.Web.Auth.Enable {
			t.Errorf("Expected web auth enabled by default")
		}

		if cfg.Web.Auth.SessionTimeout != 24*time.Hour {
			t.Errorf("Expected default session timeout 24h, got %s", cfg.Web.Auth.SessionTimeout)
		}

		if len(cfg.Web.Auth.Users) == 0 {
			t.Fatalf("Expected default auth users to be populated")
		}

		if !cfg.Web.Export.Enable {
			t.Errorf("Expected export enabled by default")
		}

		if len(cfg.Web.Export.Formats) == 0 {
			t.Errorf("Expected default export formats")
		}

		if cfg.Output.Mode != "console" {
			t.Errorf("Expected default output mode console, got %s", cfg.Output.Mode)
		}

		if cfg.Output.Silence {
			t.Errorf("Expected silence mode disabled by default")
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
					Responses: defaultResponses(),
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
					Responses: defaultResponses(),
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
					Responses: defaultResponses(),
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
					Responses: defaultResponses(),
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
					Responses: defaultResponses(),
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
					Responses: defaultResponses(),
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
					Responses: defaultResponses(),
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
					Responses: defaultResponses(),
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
		{
			name: "Missing server responses",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Path: "/",
					Responses: []ImmediateResponseConfig{},
				},
				Log: LogConfig{Level: "info"},
				Forward: ForwardConfig{MaxConcurrent: 1},
			},
			expectError: true,
			errorMsg:    "server responses configuration cannot be empty",
		},
		{
			name: "Web enabled but missing path",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Path: "/",
					Responses: defaultResponses(),
				},
				Log: LogConfig{
					Level: "info",
				},
				Forward: ForwardConfig{
					MaxConcurrent: 1,
				},
				Web: WebConfig{
					Enable:      true,
					Path:        "",
					AdminPath:   "/api",
					MaxRequests: 100,
					Auth: WebAuthConfig{
						Enable: false,
					},
				},
			},
			expectError: true,
			errorMsg:    "web path cannot be empty",
		},
		{
			name: "Web auth enabled but no users",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Path: "/",
					Responses: defaultResponses(),
				},
				Log: LogConfig{
					Level: "info",
				},
				Forward: ForwardConfig{
					MaxConcurrent: 1,
				},
				Web: WebConfig{
					Enable:      true,
					Path:        "/web",
					AdminPath:   "/api",
					MaxRequests: 100,
					Auth: WebAuthConfig{
						Enable:         true,
						SessionTimeout: time.Hour,
						Users:          []WebUserConfig{},
					},
				},
			},
			expectError: true,
			errorMsg:    "web auth requires at least one user",
		},
		{
			name: "Invalid output mode",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Path: "/",
					Responses: defaultResponses(),
				},
				Log: LogConfig{Level: "info"},
				Forward: ForwardConfig{MaxConcurrent: 1},
				Output: OutputConfig{Mode: "yaml"},
			},
			expectError: true,
			errorMsg:    "output mode",
		},
		{
			name: "Invalid path strategy mode",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Path: "/",
					Responses: defaultResponses(),
				},
				Log: LogConfig{Level: "info"},
				Forward: ForwardConfig{
					MaxConcurrent: 1,
					PathStrategy: ForwardPathStrategyConfig{Mode: "mystery"},
				},
			},
			expectError: true,
			errorMsg:    "forward path strategy mode",
		},
		{
			name: "Rewrite mode missing rules",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Path: "/",
					Responses: defaultResponses(),
				},
				Log: LogConfig{Level: "info"},
				Forward: ForwardConfig{
					MaxConcurrent: 1,
					PathStrategy: ForwardPathStrategyConfig{Mode: "rewrite"},
				},
			},
			expectError: true,
			errorMsg:    "rules cannot be empty",
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
  responses:
    - name: "custom"
      methods: ["POST"]
      path: "/test"
      status: 202
      body: '{"status":"queued"}'
      headers:
        Content-Type: application/json

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
  path_strategy:
    mode: "strip_prefix"
    strip_prefix: "/test"

output:
  mode: json
  silence: true
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
	cfg, err := LoadConfig(tmpFile.Name(), nil)
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

	if len(cfg.Server.Responses) != 1 {
		t.Fatalf("Expected 1 response rule, got %d", len(cfg.Server.Responses))
	}

	resp := cfg.Server.Responses[0]
	if resp.Status != 202 || resp.Body == "" {
		t.Errorf("Unexpected response config: %+v", resp)
	}
	if resp.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected response header Content-Type application/json, got %s", resp.Headers["Content-Type"])
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

	if cfg.Forward.PathStrategy.Mode != "strip_prefix" {
		t.Errorf("Expected path strategy strip_prefix, got %s", cfg.Forward.PathStrategy.Mode)
	}

	if cfg.Forward.PathStrategy.StripPrefix != "/test" {
		t.Errorf("Expected strip prefix '/test', got %s", cfg.Forward.PathStrategy.StripPrefix)
	}

	if cfg.Output.Mode != "json" || !cfg.Output.Silence {
		t.Errorf("Expected output json+silence true, got mode=%s silence=%v", cfg.Output.Mode, cfg.Output.Silence)
	}
}

func TestLoadConfigInvalidFile(t *testing.T) {
	// Test that non-existent configuration file should return error
	cfg, err := LoadConfig("/nonexistent/path/config.yaml", nil)
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
