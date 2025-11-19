# ReqTap - HTTP 请求调试工具

[![许可证](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Go 版本](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![构建状态](https://img.shields.io/github/actions/workflow/status/funnyzak/reqtap/release.yml)](https://github.com/funnyzak/reqtap/actions)
[![Image Size](https://img.shields.io/docker/image-size/funnyzak/reqtap)](https://hub.docker.com/r/funnyzak/reqtap/)
[![GitHub 发布](https://img.shields.io/github/v/release/funnyzak/reqtap)](https://github.com/funnyzak/reqtap/releases)

[English](README-EN.md) | **中文**

ReqTap 是一个强大的、跨平台的、零依赖命令行工具，用于即时捕获、检查和转发 HTTP 请求。它作为您的终极"请求黑洞"和"webhook 调试器"，提供无缝的 HTTP 请求分析功能。

## 使用场景

ReqTap 适用于以下场景：

- **Webhook 开发与调试** - 接收并查看来自其他系统的 HTTP 通知
- **API 接口测试和调试** - 检查客户端发送的请求数据是否正确
- **前端开发请求分析** - 分析网页 JavaScript 发送的 API 请求
- **微服务通信调试** - 监控不同服务之间的 HTTP 调用
- **网络安全审计** - 捕获可疑的网络请求进行分析
- **支付回调处理** - 接收支付宝、微信等平台的支付通知
- **自动化工作流测试** - 测试 Zapier、n8n 等自动化工具的触发器
- **HTTP 协议教学演示** - 直观展示 HTTP 请求的结构和格式
- **系统监控和故障排查** - 快速测试网络连通性和服务可用性
- **代理和网关开发** - 作为代理服务器的调试工具
- **数据收集和分析** - 接收来自各种设备或系统的数据上报

## 特性

- **可编排的即时响应** - 通过 `server.responses` 为不同路径/方法配置专属状态码、Body 和 Header，轻松模拟目标服务
- **丰富的可视化输出** - 美观的彩色终端输出，采用标准 HTTP 报文排版呈现请求行/头/体，并支持 HTTP 方法、头部和请求体的语法高亮
- **安全优先** - 智能二进制内容检测和敏感信息自动脱敏
- **异步转发** - 高性能异步请求转发到多个目标 URL
- **转发路径策略** - `append`、`strip_prefix`、`rewrite` 三种模式适配多环境 URL 差异
- **全面日志记录** - 双日志系统，支持控制台输出和结构化文件日志，带自动轮转
- **灵活配置** - 支持命令行参数、YAML 配置文件和环境变量
- **实时 Web 控制台** - 基于 Session 的仪表盘，提供 WebSocket 实时流、筛选搜索、JSON/CSV/文本导出
- **跨平台** - 单一可执行文件，原生支持 Windows、macOS 和 Linux
- **零依赖** - 自包含二进制文件，无外部运行时依赖
- **CI / 日志友好** - `--silence` 跳过富文本输出，`--json` 以结构化日志输出，易于接入管线

## 预览

### 运行预览

![运行预览](https://github.com/user-attachments/assets/725b948f-4f5e-407f-b2b2-9e4b6a0d4179)

### 实时控制台

![实时控制台](https://github.com/user-attachments/assets/353ff050-0022-4779-b2ce-2bf186cf0707)

## 快速开始

### 安装

#### 选项 1：使用 Homebrew（推荐）

macOS 用户首选的安装方式是使用 Homebrew：

```bash
# 添加 tap
brew tap funnyzak/reqtap

# 安装 reqtap
brew install reqtap

# 更新
brew update && brew upgrade reqtap
```

#### 选项 2：使用安装脚本

最简单的跨平台安装方式是使用的安装脚本：

```bash
curl -fsSL https://raw.githubusercontent.com/funnyzak/reqtap/main/scripts/install.sh | bash
```

或者下载后手动运行：

```bash
# 下载脚本
curl -fsSL https://raw.githubusercontent.com/funnyzak/reqtap/main/scripts/install.sh -o install.sh

# 运行脚本
chmod +x install.sh
./install.sh
```

脚本支持多个命令：

- `install` - 安装 ReqTap（默认）
- `update` - 更新到最新版本
- `uninstall` - 卸载 ReqTap
- `check` - 检查已安装版本和可用更新
- `list` - 列出所有可用版本

示例：

```bash
# 安装最新版本
curl -fsSL https://raw.githubusercontent.com/funnyzak/reqtap/main/scripts/install.sh | bash -s install

# 安装指定版本
curl -fsSL https://raw.githubusercontent.com/funnyzak/reqtap/main/scripts/install.sh | bash -s install -v 0.1.5

# 更新到最新版本
curl -fsSL https://raw.githubusercontent.com/funnyzak/reqtap/main/scripts/install.sh | bash -s update
```

#### 选项 3：下载预编译二进制文件

1. 访问 [发布页面](https://github.com/funnyzak/reqtap/releases)
2. 下载适合您平台的二进制文件：
   - `reqtap-linux-amd64` 用于 Linux x86_64
   - `reqtap-linux-arm64` 用于 Linux ARM64
   - `reqtap-linux-arm` 用于 Linux ARMv7
   - `reqtap-linux-ppc64le` 用于 Linux PowerPC 64 LE
   - `reqtap-linux-riscv64` 用于 Linux RISC-V 64
   - `reqtap-linux-s390x` 用于 Linux IBM Z
   - `reqtap-darwin-amd64` 用于 macOS Intel
   - `reqtap-darwin-arm64` 用于 macOS Apple Silicon
   - `reqtap-windows-amd64.exe` 用于 Windows x86_64
3. 添加可执行权限（Unix 系统）：
   ```bash
   chmod +x reqtap-*
   mv reqtap-* reqtap
   ```

#### 选项 4：使用 Docker

```bash
# 拉取最新镜像
docker pull funnyzak/reqtap:latest

# 使用默认设置运行 ReqTap
docker run -p 38888:38888 funnyzak/reqtap:latest

# 使用自定义配置运行
docker run -p 8080:38888 -v $(pwd)/config.yaml:/app/config.yaml funnyzak/reqtap:latest --config /app/config.yaml
```

#### 选项 5：从源码构建

```bash
# 克隆仓库
git clone https://github.com/funnyzak/reqtap.git
cd reqtap

# 为当前平台构建
make build

# 或直接使用 Go 构建
go build -o reqtap ./cmd/reqtap
```

### 基本使用

1. **使用默认设置启动**
   ```bash
   reqtap
   ```
   默认监听 `http://0.0.0.0:38888/reqtap`

2. **自定义端口和路径**
   ```bash
   reqtap --port 8080 --path /reqtap/
   ```

3. **启用文件日志记录**
   ```bash
   reqtap --log-file-enable --log-file-path ./reqtap.log
   ```

4. **转发到多个目标**
   ```bash
   reqtap --forward-url http://localhost:3000/webhook --forward-url https://api.example.com/ingest
   ```

## Web 控制台

当 `web.enable` 为 `true`（默认值）时，ReqTap 会自动提供一个零依赖的网页控制台，默认入口为 `http://<host>:<port>/web`，它可以：

- 使用 Session 登录控制台（默认账号：`admin/admin123`，`user/user123`，请及时修改）
- 通过 WebSocket 实时流观察最新请求
- 根据 HTTP 方法、路径、Query、头部或来源 IP 进行筛选/搜索
- 在模态窗口中查看完整的请求详情（Headers + Body）
- 一键导出当前视图为 JSON、CSV 或纯文本
- 在控制台右上角切换暗色/亮色主题，偏好会自动保存在浏览器中
- 管理员可对任一请求直接复制/下载 Request 报文、复制/下载固定 Response 报文，以及复制可直接重放的 cURL 命令

控制台使用的 API 位于可配置的 `web.admin_path`（默认 `/api`）下：

| Method | Path | 说明 |
| ------ | ---- | ---- |
| `POST` | `/api/auth/login` | 账号登录，创建 Session |
| `POST` | `/api/auth/logout` | 退出登录 |
| `GET`  | `/api/auth/me` | 获取当前用户信息 |
| `GET`  | `/api/requests` | 查询最近请求，支持 `search`、`method`、`limit`、`offset` |
| `GET`  | `/api/export` | 根据过滤条件导出 JSON/CSV/TXT |
| `GET`  | `/api/ws` | WebSocket 通道，实时推送新请求 |

通过配置文件的 `web` 段可以调整访问路径、最大缓存数量，或完全关闭 Web 控制台。

5. **使用 curl 快速测试**
   ```bash
   curl -X POST http://localhost:38888/reqtap \
     -H "Content-Type: application/json" \
     -d '{"message": "Hello, ReqTap!"}'
   ```

## 配置

### 命令行选项

```text
用法:
  reqtap [flags]

标志:
  -c, --config string              配置文件路径 (默认 "config.yaml")
  -p, --port int                   监听端口 (默认 38888)
      --path string                要监听的 URL 路径前缀 (默认 "/reqtap")
      --max-body-bytes int         单个请求体允许的最大大小（字节，0 表示无限制）(默认 10485760)
  -l, --log-level string           日志级别: trace, debug, info, warn, error, fatal, panic (默认 "info")
      --log-file-enable            启用文件日志
      --log-file-path string       日志文件路径 (默认 "./reqtap.log")
      --log-file-max-size int      单个日志文件的最大大小（MB）(默认 10)
      --log-file-max-backups int   保留的旧日志文件最大数量 (默认 5)
      --log-file-max-age int       旧日志文件的最大保留天数 (默认 30)
      --log-file-compress          是否压缩旧日志文件 (默认 true)
      --silence                    静默模式，不打印 banner 和请求详情
      --json                       输出 JSON 日志，便于 CI / 日志系统
      --body-view                  启用多格式正文展示（JSON 缩进、表单表格等）
      --body-preview-bytes int     控制台正文预览的最大字节数（超过即截断）
      --full-body                  无视预览限制，始终输出完整请求体
      --body-hex-preview           为二进制正文开启十六进制预览
      --body-hex-preview-bytes int 十六进制预览字节上限
      --body-save-binary           将二进制正文落盘保存
      --body-save-directory string 自定义二进制落盘目录（需配合 --body-save-binary）
  -f, --forward-url stringSlice    要转发请求的目标 URL
      --forward-timeout int        转发请求超时时间（秒）(默认 30)
      --forward-max-retries int    转发请求的最大重试次数 (默认 3)
      --forward-max-concurrent int 最大并发转发请求数 (默认 10)
  -h, --help                       显示帮助信息
  -v, --version                    显示版本信息
```

### 配置文件

创建 `config.yaml` 文件用于持久化配置：

```yaml
# 服务器配置
server:
  port: 38888
  path: "/reqtap"
  max_body_bytes: 10485760  # 单个请求体的最大字节数，0 表示不限制
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

# 日志配置
log:
  level: "info"  # trace, debug, info, warn, error, fatal, panic
  file_logging:
    enable: true
    path: "./reqtap.log"
    max_size_mb: 10      # 每个文件的最大大小（MB）
    max_backups: 5       # 旧日志文件的最大数量
    max_age_days: 30     # 最大保留天数
    compress: true       # 压缩旧日志文件

# 转发配置
forward:
  urls:
    - "http://localhost:3000/webhook"
    - "https://api.example.com/ingest"
  timeout: 30           # 请求超时时间（秒）
  response_header_timeout: 15  # 响应头超时时间（秒），防止上游挂起
  tls_handshake_timeout: 10    # TLS 握手超时（秒）
  expect_continue_timeout: 1   # Expect-Continue 等待时间（秒）
  max_retries: 3        # 最大重试次数
  max_concurrent: 10    # 最大并发转发数
  max_idle_conns: 200            # 最大空闲连接数
  max_idle_conns_per_host: 50    # 每主机最大空闲连接数
  max_conns_per_host: 100        # 每主机最大连接数
  idle_conn_timeout: 90          # 空闲连接超时（秒）
  tls_insecure_skip_verify: false # 是否跳过 TLS 校验（仅限测试环境）
  path_strategy:
    mode: "strip_prefix"        # append / strip_prefix / rewrite
    strip_prefix: "/reqtap"     # strip_prefix 为空时默认使用 server.path
    # rewrite 模式示例
    # rules:
    #   - name: "rewrite-service"
    #     match: "/service"
    #     replace: "/api"
    #   - name: "regex-tenant"
    #     match: "^/tenant/(.*)$"
    #     replace: "/$1"
    #     regex: true
  # Header 过滤，黑名单先于白名单；白名单非空时只转发列出的 Header
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

> **Forward 提示**
> - `urls` 可配置多个下游地址，ReqTap 会并发发送，并遵循 `timeout`、`max_retries` 等限制。
> - `path_strategy.mode` 为 `append` 时保持默认行为（直接拼接原始路径）；`strip_prefix` 会在转发前剥离监听前缀（默认使用 `server.path`）；`rewrite` 则按 `rules` 顺序执行前缀或正则改写。
> - 例如：在本例配置下，外部命中的 `/reqtap/demo` 会被裁剪成 `/demo` 后再转发到目标 URL，减少多环境路径差异。

# Web 控制台
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

# CLI 输出
output:
  mode: "console"   # console / json
  silence: false     # true 时不打印彩色输出
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
```

默认情况下会限制请求体为 10 MB，可通过 `server.max_body_bytes` 或 `--max-body-bytes` 调整，设置为 `0` 表示不做限制。

其中：

- `server.responses` 以声明式方式模拟不同的响应，支持 `path`、`path_prefix`、`methods` 组合匹配，第一条匹配即生效；`path`/`path_prefix` 必须写入包含 `server.path`（默认 `/reqtap`）的完整路径。
- `forward.path_strategy` 允许在转发阶段去除监听前缀或执行自定义重写，避免多环境回调 URL 不一致。
- `output.mode` 与 `output.silence` 分别控制彩色输出/JSON 行与静默模式，也可通过 `--json`、`--silence` 临时覆盖。
- `output.body_view` 负责多格式正文展示：开启后可自动对 JSON 缩进（含最大缩进阈值）、表单体转表格、XML/HTML 美化或剥离控制字符，并为二进制体提供十六进制预览与落盘；CLI 可用 `--body-view`、`--body-preview-bytes`、`--full-body`、`--body-hex-preview`、`--body-hex-preview-bytes`、`--body-save-binary`、`--body-save-directory` 即时覆盖相关开关及限额。

**使用配置文件：**
```bash
reqtap --config config.yaml
```

### 环境变量

所有配置选项都可以通过带有 `REQTAP_` 前缀的环境变量设置：

```bash
# 服务器设置
export REQTAP_SERVER_PORT=8080
export REQTAP_SERVER_PATH="/reqtap"
export REQTAP_SERVER_MAX_BODY_BYTES=2097152

# 日志设置
export REQTAP_LOG_LEVEL=debug
export REQTAP_LOG_FILE_ENABLE=true
export REQTAP_LOG_FILE_PATH="/var/log/reqtap.log"

# 转发设置
export REQTAP_FORWARD_URLS="http://localhost:3000/webhook,https://api.example.com/ingest"
export REQTAP_FORWARD_TIMEOUT=30

# Web 控制台
export REQTAP_WEB_ENABLE=true
export REQTAP_WEB_PATH="/console"
export REQTAP_WEB_AUTH_SESSION_TIMEOUT=12h

# 输出模式
export REQTAP_OUTPUT_MODE=json
export REQTAP_OUTPUT_SILENCE=false

# 启动 ReqTap
./reqtap
```

### 场景示例

#### Webhook 调试
```bash
# 监听 8080 端口，启用 Web 控制台，转发到本地开发服务
reqtap --port 8080 --web.enable --forward-url http://localhost:3000/webhook
```

#### 安全审计
```bash
# 详细日志模式，记录所有请求信息
reqtap --log-level debug --log-file-enable --log-file-path ./audit.log
```

#### CI / 日志管线
```bash
# 使用 JSON 输出便于解析，可搭配 --silence 禁用彩色输出
reqtap --json --forward-url https://ci.local/collector
```

### 配置优先级

配置按以下顺序加载（优先级从高到低）：

1. **命令行参数**
2. **环境变量**
3. **配置文件**
4. **默认值**

## 架构概览

ReqTap 由若干松耦合的内部包组成，每个包都负责请求生命周期中的一个阶段：

- **CLI 启动层（`cmd/reqtap`）**：基于 Cobra/Viper 组合命令行参数、环境变量与 YAML 配置，启动前完成配置校验并输出运行信息。
- **配置与日志（`internal/config`, `internal/logger`）**：`config` 统一默认值、加载顺序与约束校验；`logger` 使用 zerolog + lumberjack 在终端和彩色滚动日志之间共享一套结构化日志接口。
- **HTTP 服务层（`internal/server`）**：利用 Gorilla Mux 构建路由，`Handler` 会在读取完请求体后立即返回 200 OK，真正的处理逻辑在后台 goroutine 中异步执行。
- **请求处理流水线（`pkg/request`, `internal/printer`, `internal/web`, `internal/forwarder`）**：`RequestData` 将原始 `http.Request` 规范化；随后通过 `sync.WaitGroup` fan-out 到控制台打印、Web 控制台入库/推送以及多目标转发，实现彼此独立的消费者。
- **转发器（`internal/forwarder`）**：维持一个有界 worker 池，结合 `context.Context` 超时和指数退避重试策略，将请求复制到所有目标地址并补充 `X-ReqTap-*` 追踪头。
- **Web 控制台（`internal/web`, `internal/static`）**：包含基于环形缓冲与方法索引的 `RequestStore`、Session 登录管理、WebSocket 推送、JSON/CSV/TXT 流式导出以及内嵌前端资源，可通过 `web.path`/`web.admin_path` 在任意前缀下提供 UI 与 API。
- **可观测性**：所有组件都依赖同一个 `logger.Logger` 接口输出关键字段，便于在 CLI 与文件日志之间保持一致的调试体验。

```text
Clients --> gorilla/mux Router --> Handler --> immediate 200 OK
                           |
                           |-- ConsolePrinter (彩色输出 / 脱敏)
                           |-- Web Service (RequestStore + REST + WebSocket)
                           `-- Forwarder (worker pool + retries --> Targets)
```

### 请求生命周期

1. CLI 入口解析 flag/env/config 并创建 Logger、Forwarder、ConsolePrinter 与 Web Service 等依赖。
2. Gorilla Mux 捕获任意匹配 `server.path` 的请求，Handler 读取完整请求体后立即向客户端返回 `200 OK` 与 `ok` 文本，保证调用方不被阻塞。
3. Handler 将请求转换为 `RequestData`，记录基础信息，并把数据交给 Web Service 进行持久化（环形缓冲）以及 WebSocket 推送。
4. 控制台打印器以动态终端宽度渲染彩色输出，同时根据内置规则自动检测二进制内容并对敏感 header 做脱敏。
5. 若配置了转发地址，Forwarder 会在独立 goroutine 中并发向所有目标发送请求，遵循超时、最大并发和重试策略，遇到 4xx/5xx 会按指数退避重试并输出结构化日志。
6. 所有后台任务完成后该请求的 goroutine 才会退出，确保转发和日志写入完成，但不会影响已经返回的客户端响应。

## 从源码构建

### 前置条件

- Go 1.21 或更高版本
- Make（可选，用于构建脚本）

### 构建命令

```bash
# 为当前平台构建
make build

# 为所有平台交叉编译
make build-all

# 运行测试
make test

# 运行测试并生成覆盖率报告
make test-coverage

# 安装依赖
make deps

# 清理构建产物
make clean
```

### 手动构建

```bash
# 为当前平台构建
go build -o reqtap ./cmd/reqtap

# 为特定平台构建
GOOS=linux GOARCH=amd64 go build -o reqtap-linux-amd64 ./cmd/reqtap
GOOS=darwin GOARCH=amd64 go build -o reqtap-darwin-amd64 ./cmd/reqtap
GOOS=windows GOARCH=amd64 go build -o reqtap-windows-amd64.exe ./cmd/reqtap

# 构建时包含版本信息
go build -ldflags "-X main.version=0.1.5" -o reqtap ./cmd/reqtap
```

## 开发

### 开发环境设置

1. **克隆仓库**
   ```bash
   git clone https://github.com/funnyzak/reqtap.git
   cd reqtap
   ```

2. **安装依赖**
   ```bash
   go mod download
   make deps
   ```

3. **运行测试**
   ```bash
   make test
   ```

4. **在开发模式下运行**
   ```bash
   go run ./cmd/reqtap --log-level debug
   ```
### 项目结构

```
reqtap/
├── cmd/reqtap/main.go        # Cobra CLI 与服务器入口
├── internal/
│   ├── config/               # 配置默认值、加载与校验
│   ├── forwarder/            # 多目标转发、重试与并发控制
│   ├── logger/               # zerolog 适配器 + 可选文件日志
│   ├── printer/console.go    # 终端彩色打印与敏感信息脱敏
│   ├── server/               # Gorilla Mux 服务器和 Handler
│   ├── static/               # 内嵌 Web 控制台静态资源
│   └── web/                  # Dashboard API、WebSocket、存储、认证
├── pkg/request/request.go    # RequestData 结构与辅助函数
├── scripts/install.sh        # 安装/升级脚本
├── config.yaml.example       # 配置示例
├── Dockerfile
├── Makefile
├── README.md / README-EN.md
└── build/, logs/             # 可选的构建产物和滚动日志输出
```

---

## 许可证

本项目采用 [MIT 许可证](LICENSE)。
