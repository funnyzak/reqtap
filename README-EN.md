# ReqTap - HTTP Request Debugging Tool

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/funnyzak/reqtap)](https://github.com/funnyzak/reqtap/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/funnyzak/reqtap/build.yml)](https://github.com/funnyzak/reqtap/actions)
[![Docker Tags](https://img.shields.io/docker/v/funnyzak/reqtap?sort=semver&style=flat-square)](https://hub.docker.com/r/funnyzak/reqtap/)

**English** | [中文文档](README.md)

ReqTap is a powerful, cross-platform, zero-dependency command-line tool for instantly capturing, inspecting, and forwarding HTTP requests. It serves as your ultimate "request blackhole" and "webhook debugger" for seamless HTTP request analysis.

## Features

- **Instant Response** - Immediately returns 200 OK upon receiving requests, ensuring non-blocking client operations
- **Rich Visual Output** - Beautiful colored terminal output with highlighting for HTTP methods, headers, and request bodies
- **Security-First** - Intelligent binary content detection and automatic sensitive information redaction
- **Async Forwarding** - High-performance asynchronous request forwarding to multiple target URLs
- **Comprehensive Logging** - Dual logging system with console output and structured file logging with automatic rotation
- **Flexible Configuration** - Support for command-line arguments, YAML configuration files, and environment variables
- **Cross-Platform** - Single executable with native support for Windows, macOS, and Linux
- **Zero Dependencies** - Self-contained binary with no external runtime requirements

## Preview

```text
══════════════════════════════════════════════════════════════════════
                              ReqTap
                         HTTP Request Debugging Tool v1.0.0
══════════════════════════════════════════════════════════════════════
Listen Address: http://0.0.0.0:38888/
Forward Targets: [http://localhost:3000/webhook]
Log File: ./reqtap.log
══════════════════════════════════════════════════════════════════════

═ INCOMING REQUEST (2024-01-15T10:30:45+08:00) ═
[POST] /webhook/payment [FROM: 192.168.1.100]
User-Agent: curl/7.64.1
─ Headers ─
Content-Type: application/json
Authorization: [REDACTED]
X-Request-ID: req-123456
─ Body ─
{
  "order_id": "12345",
  "amount": 99.99,
  "currency": "USD"
}
═ END OF REQUEST ═
```

## Quick Start

### Installation

#### Option 1: Download Pre-compiled Binary (Recommended)

1. Go to the [Releases](https://github.com/funnyzak/reqtap/releases) page
2. Download the appropriate binary for your platform:
   - `reqtap-linux-amd64` for Linux
   - `reqtap-darwin-amd64` for macOS Intel
   - `reqtap-darwin-arm64` for macOS Apple Silicon
   - `reqtap-windows-amd64.exe` for Windows
3. Make it executable (Unix systems):
   ```bash
   chmod +x reqtap-*
   mv reqtap-* reqtap
   ```

#### Option 2: Using Docker

```bash
# Pull the latest image
docker pull funnyzak/reqtap:latest

# Run ReqTap with default settings
docker run -p 38888:38888 funnyzak/reqtap:latest

# Run with custom configuration
docker run -p 8080:38888 -v $(pwd)/config.yaml:/app/config.yaml funnyzak/reqtap:latest --config /app/config.yaml
```

#### Option 3: Build from Source

```bash
# Clone the repository
git clone https://github.com/funnyzak/reqtap.git
cd reqtap

# Build for current platform
make build

# Or use Go directly
go build -o reqtap ./cmd/reqtap
```

### Basic Usage

1. **Start with default settings**
   ```bash
   ./reqtap
   ```
   Listens on `http://0.0.0.0:38888/` by default

2. **Custom port and path**
   ```bash
   ./reqtap --port 8080 --path /webhook/
   ```

3. **Enable file logging**
   ```bash
   ./reqtap --log-file-enable --log-file-path ./reqtap.log
   ```

4. **Forward to multiple targets**
   ```bash
   ./reqtap --forward-url http://localhost:3000/webhook --forward-url https://api.example.com/ingest
   ```

5. **Quick test with curl**
   ```bash
   curl -X POST http://localhost:38888/webhook \
     -H "Content-Type: application/json" \
     -d '{"message": "Hello, ReqTap!"}'
   ```

## Configuration

### Command Line Options

```text
Usage:
  reqtap [flags]

Flags:
  -c, --config string              Configuration file path (default "config.yaml")
  -p, --port int                   Listen port (default 38888)
      --path string                URL path prefix to listen (default "/")
  -l, --log-level string           Log level: trace, debug, info, warn, error, fatal, panic (default "info")
      --log-file-enable            Enable file logging
      --log-file-path string       Log file path (default "./reqtap.log")
      --log-file-max-size int      Maximum size of a single log file in MB (default 10)
      --log-file-max-backups int   Maximum number of old log files to retain (default 5)
      --log-file-max-age int       Maximum retention days for old log files (default 30)
      --log-file-compress          Whether to compress old log files (default true)
  -f, --forward-url stringSlice    Target URLs to forward requests to
      --forward-timeout int        Forward request timeout in seconds (default 30)
      --forward-max-retries int    Maximum retry attempts for forwarded requests (default 3)
      --forward-max-concurrent int Maximum concurrent forward requests (default 10)
  -h, --help                       Show help information
  -v, --version                    Show version information
```

### Configuration File

Create a `config.yaml` file for persistent configuration:

```yaml
# Server Configuration
server:
  port: 38888
  path: "/"

# Logging Configuration
log:
  level: "info"  # trace, debug, info, warn, error, fatal, panic
  file_logging:
    enable: true
    path: "./reqtap.log"
    max_size_mb: 10      # Max size per file in MB
    max_backups: 5       # Max number of old log files
    max_age_days: 30     # Max retention days
    compress: true       # Compress old log files

# Forwarding Configuration
forward:
  urls:
    - "http://localhost:3000/webhook"
    - "https://api.example.com/ingest"
  timeout: 30           # Request timeout in seconds
  max_retries: 3        # Maximum retry attempts
  max_concurrent: 10    # Maximum concurrent forwards
```

**Usage with configuration file:**
```bash
./reqtap --config config.yaml
```

### Environment Variables

All configuration options can be set via environment variables with the `REQTAP_` prefix:

```bash
# Server settings
export REQTAP_SERVER_PORT=8080
export REQTAP_SERVER_PATH="/webhook"

# Logging settings
export REQTAP_LOG_LEVEL=debug
export REQTAP_LOG_FILE_ENABLE=true
export REQTAP_LOG_FILE_PATH="/var/log/reqtap.log"

# Forwarding settings
export REQTAP_FORWARD_URLS="http://localhost:3000/webhook,https://api.example.com/ingest"
export REQTAP_FORWARD_TIMEOUT=30

# Start ReqTap
./reqtap
```

### Configuration Priority

Configuration is loaded in the following order (highest priority first):

1. **Command-line arguments**
2. **Environment variables**
3. **Configuration file**
4. **Default values**

## Building from Source

### Prerequisites

- Go 1.21 or higher
- Make (optional, for build scripts)

### Build Commands

```bash
# Build for current platform
make build

# Cross-compile for all platforms
make build-all

# Run tests
make test

# Run tests with coverage
make test-coverage

# Install dependencies
make deps

# Clean build artifacts
make clean
```

### Manual Build

```bash
# Build for current platform
go build -o reqtap ./cmd/reqtap

# Build for specific platforms
GOOS=linux GOARCH=amd64 go build -o reqtap-linux-amd64 ./cmd/reqtap
GOOS=darwin GOARCH=amd64 go build -o reqtap-darwin-amd64 ./cmd/reqtap
GOOS=windows GOARCH=amd64 go build -o reqtap-windows-amd64.exe ./cmd/reqtap

# Build with version information
go build -ldflags "-X main.version=v1.0.0" -o reqtap ./cmd/reqtap
```

## Development

### Development Environment Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/funnyzak/reqtap.git
   cd reqtap
   ```

2. **Install dependencies**
   ```bash
   go mod download
   make deps
   ```

3. **Run tests**
   ```bash
   make test
   ```

4. **Run in development mode**
   ```bash
   go run ./cmd/reqtap --log-level debug
   ```

### Project Structure

```
reqtap/
├── cmd/reqtap/              # Application entry point
│   └── main.go             # Main application file
├── internal/               # Internal packages
│   ├── config/            # Configuration management
│   │   ├── config.go      # Configuration structures and loading
│   │   └── loader.go      # Configuration file loader
│   ├── server/            # HTTP server implementation
│   │   ├── server.go      # Main HTTP server
│   │   └── handler.go     # Request handlers
│   ├── printer/           # Console output formatting
│   │   ├── printer.go     # Pretty printing logic
│   │   └── colors.go      # Color schemes
│   ├── forwarder/         # Request forwarding logic
│   │   ├── forwarder.go   # Forwarding implementation
│   │   └── client.go      # HTTP client wrapper
│   └── logger/            # Logging system
│       ├── logger.go      # Logger implementation
│       └── writer.go      # Log writers
├── pkg/request/           # Request data models
│   └── request.go         # Request structure definition
├── config.yaml.example    # Configuration file template
├── Makefile              # Build scripts
├── go.mod                # Go module definition
├── go.sum                # Dependency checksums
└── docs/                 # Documentation
    └── README-zh.md      # Chinese documentation
```

## Output Examples

### Basic Request Logging

```text
═ INCOMING REQUEST (2024-01-15T10:30:45+08:00) ═
[GET] /api/users [FROM: 192.168.1.100]
User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64)
─ Headers ─
Accept: application/json
Authorization: Bearer [REDACTED]
─ Body ─
(empty)
═ END OF REQUEST ═
```

### JSON Payload with Forwarding

```text
═ INCOMING REQUEST (2024-01-15T10:35:22+08:00) ═
[POST] /webhook/github [FROM: 140.82.112.1]
User-Agent: GitHub-Hookshot/abc123
─ Headers ─
Content-Type: application/json
X-GitHub-Event: push
X-GitHub-Delivery: 12345678-1234-1234-1234-123456789012
─ Body ─
{
  "ref": "refs/heads/main",
  "repository": {
    "name": "reqtap",
    "full_name": "funnyzak/reqtap"
  },
  "pusher": {
    "name": "username",
    "email": "user@example.com"
  },
  "commits": [
    {
      "id": "abc123",
      "message": "Update README",
      "author": {
        "name": "Developer",
        "email": "dev@example.com"
      }
    }
  ]
}
═ END OF REQUEST ═

→ Forwarding to http://localhost:3000/webhook... ✓
→ Forwarding to https://api.example.com/ingest... ✓
```

### Binary Content Detection

```text
═ INCOMING REQUEST (2024-01-15T10:40:15+08:00) ═
[POST] /upload [FROM: 192.168.1.100]
User-Agent: curl/7.64.1
─ Headers ─
Content-Type: application/octet-stream
Content-Length: 1024
─ Body ─
[BINARY CONTENT - 1024 bytes skipped]
═ END OF REQUEST ═
```

---

## License

Under the [MIT License](LICENSE).