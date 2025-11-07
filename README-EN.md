# ReqTap - HTTP Request Debugging Tool

[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![Build Status](https://img.shields.io/github/actions/workflow/status/funnyzak/reqtap/release.yml)](https://github.com/funnyzak/reqtap/actions)
[![Image Size](https://img.shields.io/docker/image-size/funnyzak/reqtap)](https://hub.docker.com/r/funnyzak/reqtap/)
[![GitHub Release](https://img.shields.io/github/v/release/funnyzak/reqtap)](https://github.com/funnyzak/reqtap/releases)

**English** | [中文文档](README.md)

ReqTap is a powerful, cross-platform, zero-dependency command-line tool for instantly capturing, inspecting, and forwarding HTTP requests. It serves as your ultimate "request blackhole" and "webhook debugger" for seamless HTTP request analysis.

## Features

- **Instant Response** - Immediately returns 200 OK upon receiving requests, ensuring non-blocking client operations
- **Rich Visual Output** - Beautiful colored terminal output that renders captured data in standard HTTP message layout with highlighting for methods, headers, and bodies
- **Security-First** - Intelligent binary content detection and automatic sensitive information redaction
- **Async Forwarding** - High-performance asynchronous request forwarding to multiple target URLs
- **Comprehensive Logging** - Dual logging system with console output and structured file logging with automatic rotation
- **Flexible Configuration** - Support for command-line arguments, YAML configuration files, and environment variables
- **Realtime Web Console** - Session-based dashboard with WebSocket streaming, filtering/search and one-click JSON/CSV/TXT export
- **Cross-Platform** - Single executable with native support for Windows, macOS, and Linux
- **Zero Dependencies** - Self-contained binary with no external runtime requirements

## Preview

### Running

![Running Preview](https://github.com/user-attachments/assets/725b948f-4f5e-407f-b2b2-9e4b6a0d4179)

### Real-time Console

![Real-time Console](https://github.com/user-attachments/assets/353ff050-0022-4779-b2ce-2bf186cf0707)

## Quick Start

### Installation

#### Option 1: Using Homebrew (Recommended)

The easiest way to install ReqTap on macOS is using Homebrew:

```bash
# Add the tap
brew tap funnyzak/reqtap

# Install reqtap
brew install reqtap

# Upgrade reqtap
brew update && brew upgrade reqtap
```

#### Option 2: Using Installation Script

The easiest cross-platform way to install ReqTap is using installation script:

```bash
curl -fsSL https://raw.githubusercontent.com/funnyzak/reqtap/main/scripts/install.sh | bash
```

Or download and run manually:

```bash
# Download the script
curl -fsSL https://raw.githubusercontent.com/funnyzak/reqtap/main/scripts/install.sh -o install.sh

# Run it
chmod +x install.sh
./install.sh
```

The script supports multiple commands:

- `install` - Install ReqTap (default)
- `update` - Update to the latest version
- `uninstall` - Uninstall ReqTap
- `check` - Check installed version and available updates
- `list` - List all available versions

Examples:

```bash
# Install latest version
curl -fsSL https://raw.githubusercontent.com/funnyzak/reqtap/main/scripts/install.sh | bash -s install

# Install specific version
curl -fsSL https://raw.githubusercontent.com/funnyzak/reqtap/main/scripts/install.sh | bash -s install -v 0.1.5

# Update to latest version
curl -fsSL https://raw.githubusercontent.com/funnyzak/reqtap/main/scripts/install.sh | bash -s update
```

#### Option 3: Download Pre-compiled Binary

1. Go to the [Releases](https://github.com/funnyzak/reqtap/releases) page
2. Download the appropriate binary for your platform:
   - `reqtap-linux-amd64` for Linux x86_64
   - `reqtap-linux-arm64` for Linux ARM64
   - `reqtap-linux-arm` for Linux ARMv7
   - `reqtap-linux-ppc64le` for Linux PowerPC 64 LE
   - `reqtap-linux-riscv64` for Linux RISC-V 64
   - `reqtap-linux-s390x` for Linux IBM Z
   - `reqtap-darwin-amd64` for macOS Intel
   - `reqtap-darwin-arm64` for macOS Apple Silicon
   - `reqtap-windows-amd64.exe` for Windows x86_64
3. Make it executable (Unix systems):
   ```bash
   chmod +x reqtap-*
   mv reqtap-* reqtap
   ```

#### Option 4: Using Docker

```bash
# Pull the latest image
docker pull funnyzak/reqtap:latest

# Run ReqTap with default settings
docker run -p 38888:38888 funnyzak/reqtap:latest

# Run with custom configuration
docker run -p 8080:38888 -v $(pwd)/config.yaml:/app/config.yaml funnyzak/reqtap:latest --config /app/config.yaml
```

#### Option 5: Build from Source

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
   reqtap
   ```
   Listens on `http://0.0.0.0:38888/` by default

2. **Custom port and path**
   ```bash
   reqtap --port 8080 --path /reqtap/
   ```

3. **Enable file logging**
   ```bash
   reqtap --log-file-enable --log-file-path ./reqtap.log
   ```

4. **Forward to multiple targets**
   ```bash
   reqtap --forward-url http://localhost:3000/webhook --forward-url https://api.example.com/ingest
   ```

## Web Dashboard

ReqTap ships with a zero-dependency web console that is enabled by default. Once the server is running you can open `http://<host>:<port>/web` to:

- Log in with session-based authentication (default accounts: `admin/admin123`, `user/user123`)
- Watch incoming requests in real-time via WebSocket streaming
- Filter/search by HTTP method, path, query, headers, or origin IP
- Inspect full request details (headers + body) in a modal panel
- Export the current view as JSON, CSV, or plain text with a single click
- (Admins only) Copy/download the full request payload, copy/download the default response payload, and grab a ready-to-run cURL command for any request

APIs powering the dashboard live under the configurable `web.admin_path` (defaults to `/api`):

| Method | Path | Description |
| ------ | ---- | ----------- |
| `POST` | `/api/auth/login` | Authenticate and create a session cookie |
| `POST` | `/api/auth/logout` | Invalidate the current session |
| `GET`  | `/api/auth/me` | Retrieve current user info |
| `GET`  | `/api/requests` | List recent requests with optional `search`, `method`, `limit`, `offset` |
| `GET`  | `/api/export` | Export filtered requests as JSON/CSV/TXT |
| `GET`  | `/api/ws` | WebSocket stream broadcasting every new request |

All paths are fully configurable through the `web` section of `config.yaml`, so the dashboard can be mounted under any prefix or disabled entirely.

5. **Quick test with curl**
   ```bash
   curl -X POST http://localhost:38888/reqtap \
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

# Web Console Configuration
web:
  enable: true
  path: "/web"
  admin_path: "/api"
  max_requests: 500
  auth:
    enable: true
    session_timeout: 24h
    users:
      - username: "admin"
        password: "admin123"
        role: "admin"
      - username: "user"
        password: "user123"
        role: "viewer"
  export:
    enable: true
    formats: ["json", "csv", "txt"]
```

**Usage with configuration file:**
```bash
reqtap --config config.yaml
```

### Environment Variables

All configuration options can be set via environment variables with the `REQTAP_` prefix:

```bash
# Server settings
export REQTAP_SERVER_PORT=8080
export REQTAP_SERVER_PATH="/reqtap"

# Logging settings
export REQTAP_LOG_LEVEL=debug
export REQTAP_LOG_FILE_ENABLE=true
export REQTAP_LOG_FILE_PATH="/var/log/reqtap.log"

# Forwarding settings
export REQTAP_FORWARD_URLS="http://localhost:3000/webhook,https://api.example.com/ingest"
export REQTAP_FORWARD_TIMEOUT=30

# Web console settings
export REQTAP_WEB_ENABLE=true
export REQTAP_WEB_PATH="/console"
export REQTAP_WEB_AUTH_SESSION_TIMEOUT=12h

# Start ReqTap
./reqtap
```

### Configuration Priority

Configuration is loaded in the following order (highest priority first):

1. **Command-line arguments**
2. **Environment variables**
3. **Configuration file**
4. **Default values**

## Architecture

ReqTap is split into several loosely coupled internal packages, each responsible for a clear portion of the request lifecycle:

- **CLI bootstrap (`cmd/reqtap`)** – Cobra/Viper combine command-line flags, environment variables, and YAML files, validate the final config, and print a startup banner before the server launches.
- **Configuration & logging (`internal/config`, `internal/logger`)** – `config` owns defaults, merging rules, and validation; `logger` wraps zerolog + lumberjack so both the terminal and the rotating log file share the same structured output API.
- **HTTP service layer (`internal/server`)** – A Gorilla Mux router receives traffic, and the `Handler` returns 200 OK as soon as the body is read, while the heavy work continues inside background goroutines.
- **Request processing pipeline (`pkg/request`, `internal/printer`, `internal/web`, `internal/forwarder`)** – `RequestData` normalizes the raw `http.Request`; a `sync.WaitGroup` then fans out to console printing, dashboard persistence/WebSocket streaming, and multi-target forwarding.
- **Forwarder (`internal/forwarder`)** – Maintains a bounded worker pool, applies context timeouts plus exponential backoff retries, mirrors headers that matter, and injects `X-ReqTap-*` tracing headers for every target.
- **Web console (`internal/web`, `internal/static`)** – Includes a ring-buffer `RequestStore`, session-based auth manager, WebSocket hub, JSON/CSV/TXT export helpers, and embedded frontend assets that can be mounted under any `web.path`/`web.admin_path` combination.
- **Observability** – Every component logs through the shared `logger.Logger` interface so troubleshooting looks identical in the terminal and in file logs.

```text
Clients --> gorilla/mux Router --> Handler --> immediate 200 OK
                           |
                           |-- ConsolePrinter (colorized logging / redaction)
                           |-- Web Service (RequestStore + REST + WebSocket)
                           `-- Forwarder (worker pool + retries --> Targets)
```

### Request lifecycle

1. The CLI entrypoint resolves flags/env/config, then constructs the Logger, Forwarder, ConsolePrinter, and optional Web Service before starting the HTTP server.
2. Gorilla Mux captures any request under `server.path`. The handler reads the full body, closes it, and instantly sends `200 OK` with `ok` so clients never block on downstream work.
3. The handler converts the request into `RequestData`, emits a structured log, and hands the record to the Web Service, which stores it in a ring buffer and notifies WebSocket subscribers.
4. The console printer renders a width-aware, colorized view, detects binary payloads, and automatically redacts sensitive headers (Authorization, Cookie, etc.).
5. If forwarding is configured, the forwarder concurrently POSTs the payload to every target obeying timeout, concurrency, and retry limits. 4xx/5xx responses trigger exponential backoff retries and detailed logs.
6. The goroutine waits for printing/storage/forwarding to finish via `WaitGroup`, then exits. The client response is unaffected because it was already flushed in step 2.

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
go build -ldflags "-X main.version=0.1.5" -o reqtap ./cmd/reqtap
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
├── cmd/reqtap/main.go        # Cobra CLI & server entrypoint
├── internal/
│   ├── config/               # Defaults, loading, validation
│   ├── forwarder/            # Multi-target forwarding, retries, worker pool
│   ├── logger/               # Zerolog adapter + optional file logger
│   ├── printer/console.go    # Colorized terminal output & redaction rules
│   ├── server/               # Gorilla Mux server and handler wiring
│   ├── static/               # Embedded web console assets
│   └── web/                  # Dashboard REST API, WebSocket, store, auth
├── pkg/request/request.go    # RequestData model & helpers
├── scripts/install.sh        # Install/update script
├── config.yaml.example       # Configuration example
├── Dockerfile
├── Makefile
├── README.md / README-EN.md
└── build/, logs/             # Optional build artifacts and rolling logs
```

## License

Under the [MIT License](LICENSE).
