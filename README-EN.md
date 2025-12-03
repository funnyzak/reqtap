# ReqTap - HTTP Request Debugging Tool

[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![Build Status](https://img.shields.io/github/actions/workflow/status/funnyzak/reqtap/release.yml)](https://github.com/funnyzak/reqtap/actions)
[![Image Size](https://img.shields.io/docker/image-size/funnyzak/reqtap)](https://hub.docker.com/r/funnyzak/reqtap/)
[![GitHub Release](https://img.shields.io/github/v/release/funnyzak/reqtap)](https://github.com/funnyzak/reqtap/releases)

**English** | [中文文档](README.md)

ReqTap is a cross-platform, zero-external-dependency HTTP request capture and debugging platform. One self-contained binary ships the CLI collector, embedded SQLite storage, and a WebSocket-enabled dashboard—perfect for local development, containers, CI pipelines, or edge nodes where you need to inspect, replay, and forward HTTP calls instantly.

With ReqTap you can:

- Inspect live HTTP conversations from the terminal with colorized output while still emitting machine-readable JSON logs.
- Persist every request into SQLite, filter/search/export them, and replay payloads as needed.
- Subscribe to live traffic from the Web UI or REST APIs, or generate JSON/CSV/TXT snapshots for audits.
- Forward requests to downstream services with programmable path rewriting, retries, and tracing headers.

## Use Cases

- **Webhook & automation debugging** – Point GitHub, Stripe, Zapier, n8n, etc. to ReqTap and diff payloads safely.
- **Polyglot API development** – Funnel mobile/web/IoT traffic through a single capture point before forwarding to staging.
- **Microservice & gateway troubleshooting** – Freeze real requests before they reach unstable services, then replay/export during incidents.
- **Security / zero-trust auditing** – Keep the exposed surface tiny; every inbound request is persisted for offline forensics.
- **Education & workshops** – Demonstrate HTTP anatomy with CLI + Web UI views in minutes.
- **DevOps / SRE diagnostics** – Run inside CI, GitHub Actions, or Kubernetes Jobs with `--json` logs for single-shot captures.
- **Device / SDK playback** – When upstreams are offline, emulate endpoints via programmable responses while persisting real payloads.

## Key Features

- **Lightweight deployment** – Single binary + embedded SQLite (WAL + busy-timeout) means no external DBs or queues.
- **Built-in persistence** – `storage.max_records` / `storage.retention` enforce count/time pruning, and exports are available as JSON/CSV/Text.
- **Programmable mock responses** – `server.responses` script per-method/path status codes, bodies, and headers to emulate dependencies.
- **Readable CLI output** – Runewidth-aware layout, binary detection, sensitive-header redaction, optional hex preview or disk dump.
- **Concurrent forwarding** – Worker pool with timeouts, retries, and path strategies (`append`/`strip_prefix`/`rewrite`).
- **Web console + WebSocket** – Session auth, dark mode, filters, detail modals, and batch export backed by the same storage engine.
- **Unified configuration** – Cobra + Viper merge flags, env vars, and YAML; startup banners/logs echo the effective settings.
- **Logging & audits** – Zerolog JSON plus lumberjack rotation; `--silence`/`--json` keep CI/log pipelines happy.
- **Cross-platform releases** – macOS/Linux/Windows binaries, Docker images, Homebrew tap, and install scripts.
- **Security-conscious defaults** – Header black/whitelists, binary-body suppression, and export-only admin APIs for read-only integrations.
- **Full localization** – `output.locale`/`--locale` switch the CLI language, while the web console auto-detects the browser locale and offers a drop-down for instant toggling; Built-in support for English, Simplified Chinese, Japanese, Korean, French and Russian.

## Preview

### Running

![运行预览](https://github.com/user-attachments/assets/1ecf5004-6167-44db-b37e-ac841645afcc)

### Real-time Console

![实时控制台](https://github.com/user-attachments/assets/a7a4d8b5-ce4e-40aa-8356-3f5cb6633ade)

![实时控制台](https://github.com/user-attachments/assets/524c95c0-649c-4558-bf6c-756a6cd42e10)


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
   Listens on `http://0.0.0.0:38888/reqtap` by default

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

5. **Customize persistence strategy**
   ```bash
   reqtap --storage-path /var/lib/reqtap/requests.db --storage-max-records 50000 --storage-retention 168h
   ```
   The startup banner and structured logs will echo the effective storage settings so you can audit environments quickly.

## Web Dashboard

ReqTap ships with a zero-dependency web console that is enabled by default. Once the server is running you can open `http://<host>:<port>/web` to:

- Log in with session-based authentication (default accounts: `admin/admin123`, `user/user123`)
- Watch incoming requests in real-time via WebSocket streaming
- Filter/search by HTTP method, path, query, headers, or origin IP
- Inspect full request details (headers + body) in a modal panel
- Export the current view as JSON, CSV, or plain text with a single click
- Toggle between dark and light themes from the header switch; the preference is persisted locally per browser
- Access the same dark/light switch right on the login page so the experience is consistent before entering the console


### Localization

- **CLI output** – set `output.locale` (or pass `--locale en/zh-CN` at startup) to switch terminal language. Missing translations automatically fall back to English.
- **Web dashboard** – control the initial language via `web.default_locale` and expose multiple options through `web.supported_locales`. The top-right selector lets users switch instantly without reloading, and the choice is stored in `localStorage`.
- **Custom languages** – drop an additional `locales/<lang>.json` file under `internal/static/locales` (or the extracted static assets) using frontend-specific key structures. Only the differing strings are required—any gaps fall back to English so the UI remains complete.

#### Supported Languages and Configuration

The binary ships with six locales: `en`, `zh-CN`, `ja`, `ko`, `fr`, and `ru`. CLI and Web share the same dictionaries, and you can tune the behavior with the knobs below:

| Component | Entry point | Default | Notes |
| --------- | ----------- | ------- | ----- |
| CLI output | `output.locale` / `--locale` | `en` | Switches terminal prompts right at startup; command-line flags always override config files. |
| Web default locale | `web.default_locale` | `en` | Controls the language used for the very first render; compatible browsers can still override via auto-detection if the locale is supported. |
| Web supported locales | `web.supported_locales` | `[en, zh-CN, ja, ko, fr, ru]` | Defines the drop-down list in the UI. |

Sample configuration:

```yaml
output:
  locale: "zh-CN"

web:
  default_locale: "zh-CN"
  supported_locales:
    - zh-CN
    - en
    - ja
```

#### Localization Workflow

1. **Naming convention** –
   - CLI translations live in `pkg/i18n/locales/<lang>.yaml`, using `cli.*` namespace (e.g., `cli.summary.title`).
   - Web translations in `internal/static/locales/<lang>.json`, using frontend-specific key structures (e.g., `detail.meta.request_id`).
   - These subsystems use separate namespaces to avoid conflicts.
2. **Add a new locale** – copy the English template, fill only the strings you need, and list the locale code inside `web.supported_locales`/`web.default_locale` and any relevant `output.locale` defaults.
3. **Verify** – run `go test ./pkg/i18n ./internal/printer` for CLI coverage, then start `make dev` (or build) and switch languages in the browser to confirm the UI.
4. **Document** – whenever you add or rename keys, update this section so other contributors know how to keep translations in sync.
- Use the revamped detail modal tools to copy headers/body independently, flip between wrapped/scrollable layouts, and switch raw/pretty JSON views with a single click
- Enjoy the redesigned layout where the header, stats, and filter toolbar stay put while only the main request list scrolls, making long sessions easier to navigate
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
      --path string                URL path prefix to listen (default "/reqtap")
      --max-body-bytes int         Maximum allowed request body size in bytes (0 for unlimited) (default 10485760)
  -l, --log-level string           Log level: trace, debug, info, warn, error, fatal, panic (default "info")
      --log-file-enable            Enable file logging
      --log-file-path string       Log file path (default "./reqtap.log")
      --log-file-max-size int      Maximum size of a single log file in MB (default 10)
      --log-file-max-backups int   Maximum number of old log files to retain (default 5)
      --log-file-max-age int       Maximum retention days for old log files (default 30)
      --log-file-compress          Whether to compress old log files (default true)
      --silence                    Suppress banner and colorful request output
      --json                       Emit JSON lines for machine-readable pipelines
      --body-view                  Enable structured body formatting (JSON pretty, form tables, etc.)
      --body-preview-bytes int     Maximum bytes to preview before truncating the console body output
      --full-body                  Ignore preview limits and always print the complete body
      --body-hex-preview           Enable hexadecimal preview for binary bodies
      --body-hex-preview-bytes int Limit hexadecimal preview bytes
      --body-save-binary           Persist binary request bodies to disk
      --body-save-directory string Directory used when saving binary bodies (requires --body-save-binary)
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
  path: "/reqtap"
  max_body_bytes: 10485760  # Max request body size in bytes, 0 disables the limit
  responses:
    - name: "demo-json"
      methods: ["POST"]
      path_prefix: "/reqtap/demo"
      status: 202
      body: '{"status":"queued"}'
      headers:
        Content-Type: application/json
    - name: "default-ok"
      status: 200
      body: "ok"
      headers:
        Content-Type: text/plain

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
  response_header_timeout: 15  # Timeout for response headers (seconds)
  tls_handshake_timeout: 10    # TLS handshake timeout (seconds)
  expect_continue_timeout: 1   # Expect-Continue wait time (seconds)
  max_retries: 3        # Maximum retry attempts
  max_concurrent: 10    # Maximum concurrent forwards
  max_idle_conns: 200            # Max idle connections
  max_idle_conns_per_host: 50    # Max idle connections per host
  max_conns_per_host: 100        # Max connections per host
  idle_conn_timeout: 90          # Idle connection timeout (seconds)
  tls_insecure_skip_verify: false # Skip TLS verification (test only)
  path_strategy:
    mode: "strip_prefix"        # append / strip_prefix / rewrite
    strip_prefix: "/reqtap"     # Defaults to server.path when empty
    # rules:
    #   - name: "rewrite-service"
    #     match: "/service"
    #     replace: "/api"
    #   - name: "regex-tenant"
    #     match: "^/tenant/(.*)$"
    #     replace: "/$1"
    #     regex: true
  # Header filtering. Blacklist is applied first; when whitelist is non-empty, only listed headers are forwarded
  header_blacklist:
    - "host"
    - "connection"
    - "keep-alive"
    - "proxy-authenticate"
    - "proxy-authorization"
    - "te"
    - "trailers"
    - "transfer-encoding"
    - "upgrade"
    - "content-length"
  header_whitelist: []

> **Forwarding tips**
> - Populate `urls` with one or more downstream endpoints; ReqTap fans out to each while honoring `timeout`, `max_retries`, and concurrency limits.
> - With `path_strategy.mode=append` we simply stick the captured path onto the target URL. `strip_prefix` removes your listener prefix (defaults to `server.path`), and `rewrite` lets you define ordered prefix or regex replacements.
> - In this example a request entering on `/reqtap/demo` will be trimmed to `/demo` before forwarding, removing environment-specific prefixes.
> - See “Path Strategy Deep Dive” below for detailed configuration-to-forwarding walkthroughs.

#### Path Strategy Deep Dive

Using the configuration above (`server.path=/reqtap`, `forward.urls` pointing at `http://localhost:3000/webhook` and `https://api.example.com/ingest`), ReqTap validates the incoming path against the listener, transforms it according to `path_strategy`, then appends the result to every downstream URL before applying timeout, retry, and concurrency controls. Each mode behaves as follows:

- **Shared listener, downstream already has a prefix (append)**  
  A request such as `POST https://demo.test/reqtap/webhooks/payment` keeps the original `/reqtap/webhooks/payment` path. It therefore becomes `http://localhost:3000/webhook/reqtap/webhooks/payment` and `https://api.example.com/ingest/reqtap/webhooks/payment` when forwarded, which is ideal when downstream APIs are already namespaced and you just need to mirror what the client sent.
- **Different prefixes per environment (strip_prefix)**  
  Configure `mode=strip_prefix` with `strip_prefix=/reqtap` (or omit it to fall back to `server.path`). When `GET /reqtap/demo/health` arrives, ReqTap removes `/reqtap`, forwards `/demo/health`, and the downstream sees `https://api.example.com/ingest/demo/health`. Local tooling can therefore keep the canonical `/reqtap/*` entry point while production services only see their native paths.
- **Tenant or version rewrites (rewrite)**  
  Set `mode=rewrite` and declare ordered `rules`. Example rules: `match: "/service"` → `replace: "/api"`, and `match: "^/tenant/(.*)$"`, `replace: "/$1"`, `regex: true`. A request to `/reqtap/tenant/acme/orders` first (optionally) strips `/reqtap`, then the second rule rewrites it to `/acme/orders`, so the final forward becomes `https://api.example.com/ingest/acme/orders`. Rules run sequentially, and unmatched paths fall back to the last successful transformation, which makes it easy to migrate legacy routes alongside new tenants or API versions.

Regardless of the selected mode, the resulting path is cloned across all `forward.urls`, guaranteeing consistent normalization even while ReqTap fans out to multiple downstream services.

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

# CLI output
output:
  mode: "console"   # console / json
  silence: false     # true disables banner/printer output
  body_view:
    enable: false
    max_preview_bytes: 32768
    full_body: false
    json:
      enable: true
      pretty: true
      max_indent_bytes: 131072
    form:
      enable: true
    xml:
      enable: true
      pretty: true
      strip_control: true
    html:
      enable: true
      pretty: false
      strip_control: true
    binary:
      hex_preview_enable: false
      hex_preview_bytes: 256
      save_to_file: false
      save_directory: ""

# Persistent storage
storage:
  driver: "sqlite"        # only sqlite is supported today
  path: "./data/reqtap.db" # change to an absolute path if preferred
  max_records: 100000       # cap retained rows (0 = unlimited)
  retention: 0s             # optional time-based pruning, e.g. "168h"

> **Storage tips**
> - The embedded SQLite backend runs in WAL mode with a busy timeout, so a single binary works on macOS/Linux/Windows/containers without external services.
> - Combine `max_records` and `retention` to keep disk usage predictable: aged-out rows are purged first, then the remainder is trimmed by count.
> - Override at runtime with `--storage-driver`, `--storage-path`, `--storage-max-records`, or `--storage-retention`; the startup banner logs the effective settings.
> - The legacy `web.max_requests` setting no longer controls retention—use the new `storage.max_records`/`storage.retention` knobs instead.
```

By default the request body size is capped at 10 MB. Adjust `server.max_body_bytes` or pass `--max-body-bytes` to change it; set the value to `0` to remove the limit entirely.

Highlights:

- `server.responses` lets you simulate downstream services with per-path/method status, body, and headers; remember that `path`/`path_prefix` must include the full `server.path` (default `/reqtap`).
- `forward.path_strategy` normalizes forwarded paths (append, strip prefix, rewrite rules).
- `output.mode`/`output.silence` map to the `--json`/`--silence` switches for machine-readable pipelines.
- `output.body_view` powers the smart console renderer. Once enabled it prettifies JSON (with a maximum indent budget), turns form bodies into aligned tables, sanitizes XML/HTML, and offers binary helpers such as hex previews and disk persistence. Use `--body-view`, `--body-preview-bytes`, `--full-body`, `--body-hex-preview`, `--body-hex-preview-bytes`, `--body-save-binary`, and `--body-save-directory` for quick overrides.

**Usage with configuration file:**
```bash
reqtap --config config.yaml
```

### Use Case Examples

#### Webhook Debugging
```bash
# Listen on port 8080, enable web console, forward to local dev service
reqtap --port 8080 --web.enable --forward-url http://localhost:3000/webhook
```


#### Security Auditing
```bash
# Detailed logging mode to record all request information
reqtap --log-level debug --log-file-enable --log-file-path ./audit.log
```

#### CI / Logging Pipelines
```bash
# Emit newline-delimited JSON and suppress rich TTY output
reqtap --json --silence --forward-url https://ci.internal/hooks
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
- **Request processing pipeline (`pkg/request`, `internal/printer`, `internal/web`, `internal/forwarder`)** – `RequestData` normalizes the raw `http.Request`; a `sync.WaitGroup` then fans out to console printing, SQLite-backed persistence/WebSocket streaming, and multi-target forwarding.
- **Persistent storage (`internal/storage`)** – Provides a unified `storage.Store` interface with an embedded SQLite backend (WAL + busy timeout) that handles inserts, filtering/pagination, and retention/max-record pruning without extra services.
- **Forwarder (`internal/forwarder`)** – Maintains a bounded worker pool, applies context timeouts plus exponential backoff retries, mirrors headers that matter, and injects `X-ReqTap-*` tracing headers for every target.
- **Web console (`internal/web`, `internal/static`)** – Reuses `storage.Store` for history APIs, offers session-based auth, a WebSocket hub, JSON/CSV/TXT streaming exporters, and ships an embedded frontend so any `web.path`/`web.admin_path` pair can host the UI.
- **Observability** – Every component logs through the shared `logger.Logger` interface so troubleshooting looks identical in the terminal and in file logs.

```text
Clients --> gorilla/mux Router --> Handler --> immediate 200 OK
                           |
                           |-- ConsolePrinter (colorized logging / redaction)
                           |-- Storage.Store (SQLite WAL + filters/export)
                           |-- Web Service (REST + WebSocket)
                           `-- Forwarder (worker pool + retries --> Targets)
```

### Request lifecycle

1. The CLI entrypoint resolves flags/env/config, then constructs the Logger, Forwarder, ConsolePrinter, and optional Web Service before starting the HTTP server.
2. Gorilla Mux captures any request under `server.path`. The handler reads the full body, closes it, and instantly sends `200 OK` with `ok` so clients never block on downstream work.
3. The handler converts the request into `RequestData`, emits a structured log, persists it through `storage.Store`, and forwards the stored record to WebSocket subscribers.
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
