# ReqTap - HTTP 请求调试工具

[![许可证](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Go 版本](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![构建状态](https://img.shields.io/github/actions/workflow/status/funnyzak/reqtap/release.yml)](https://github.com/funnyzak/reqtap/actions)
[![Image Size](https://img.shields.io/docker/image-size/funnyzak/reqtap)](https://hub.docker.com/r/funnyzak/reqtap/)
[![GitHub 发布](https://img.shields.io/github/v/release/funnyzak/reqtap)](https://github.com/funnyzak/reqtap/releases)

[English](README-EN.md) | **中文**

ReqTap 是一个强大的、跨平台的、零依赖命令行工具，用于即时捕获、检查和转发 HTTP 请求。它作为您的终极"请求黑洞"和"webhook 调试器"，提供无缝的 HTTP 请求分析功能。

## 特性

- **即时响应** - 接收请求后立即返回 200 OK，确保客户端操作不阻塞
- **丰富的可视化输出** - 美观的彩色终端输出，支持 HTTP 方法、头部和请求体的语法高亮
- **安全优先** - 智能二进制内容检测和敏感信息自动脱敏
- **异步转发** - 高性能异步请求转发到多个目标 URL
- **全面日志记录** - 双日志系统，支持控制台输出和结构化文件日志，带自动轮转
- **灵活配置** - 支持命令行参数、YAML 配置文件和环境变量
- **跨平台** - 单一可执行文件，原生支持 Windows、macOS 和 Linux
- **零依赖** - 自包含二进制文件，无外部运行时依赖

## 运行预览

![运行预览](https://github.com/user-attachments/assets/72b7a39b-45e5-4527-979a-b5e122d9e400)

## 快速开始

### 安装

#### 选项 1：使用安装脚本（推荐）

最简单的安装方式是使用我们的安装脚本：

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

#### 选项 2：下载预编译二进制文件

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

#### 选项 3：使用 Docker

```bash
# 拉取最新镜像
docker pull funnyzak/reqtap:latest

# 使用默认设置运行 ReqTap
docker run -p 38888:38888 funnyzak/reqtap:latest

# 使用自定义配置运行
docker run -p 8080:38888 -v $(pwd)/config.yaml:/app/config.yaml funnyzak/reqtap:latest --config /app/config.yaml
```

#### 选项 4：从源码构建

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
   ./reqtap
   ```
   默认监听 `http://0.0.0.0:38888/`

2. **自定义端口和路径**
   ```bash
   ./reqtap --port 8080 --path /webhook/
   ```

3. **启用文件日志记录**
   ```bash
   ./reqtap --log-file-enable --log-file-path ./reqtap.log
   ```

4. **转发到多个目标**
   ```bash
   ./reqtap --forward-url http://localhost:3000/webhook --forward-url https://api.example.com/ingest
   ```

5. **使用 curl 快速测试**
   ```bash
   curl -X POST http://localhost:38888/webhook \
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
      --path string                要监听的 URL 路径前缀 (默认 "/")
  -l, --log-level string           日志级别: trace, debug, info, warn, error, fatal, panic (默认 "info")
      --log-file-enable            启用文件日志
      --log-file-path string       日志文件路径 (默认 "./reqtap.log")
      --log-file-max-size int      单个日志文件的最大大小（MB）(默认 10)
      --log-file-max-backups int   保留的旧日志文件最大数量 (默认 5)
      --log-file-max-age int       旧日志文件的最大保留天数 (默认 30)
      --log-file-compress          是否压缩旧日志文件 (默认 true)
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
  path: "/"

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
  max_retries: 3        # 最大重试次数
  max_concurrent: 10    # 最大并发转发数
```

**使用配置文件：**
```bash
./reqtap --config config.yaml
```

### 环境变量

所有配置选项都可以通过带有 `REQTAP_` 前缀的环境变量设置：

```bash
# 服务器设置
export REQTAP_SERVER_PORT=8080
export REQTAP_SERVER_PATH="/webhook"

# 日志设置
export REQTAP_LOG_LEVEL=debug
export REQTAP_LOG_FILE_ENABLE=true
export REQTAP_LOG_FILE_PATH="/var/log/reqtap.log"

# 转发设置
export REQTAP_FORWARD_URLS="http://localhost:3000/webhook,https://api.example.com/ingest"
export REQTAP_FORWARD_TIMEOUT=30

# 启动 ReqTap
./reqtap
```

### 配置优先级

配置按以下顺序加载（优先级从高到低）：

1. **命令行参数**
2. **环境变量**
3. **配置文件**
4. **默认值**

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
├── cmd/reqtap/              # 应用程序入口点
│   └── main.go             # 主应用程序文件
├── internal/               # 内部包
│   ├── config/            # 配置管理
│   │   ├── config.go      # 配置结构和加载
│   │   └── loader.go      # 配置文件加载器
│   ├── server/            # HTTP 服务器实现
│   │   ├── server.go      # 主 HTTP 服务器
│   │   └── handler.go     # 请求处理器
│   ├── printer/           # 控制台输出格式化
│   │   ├── printer.go     # 美化打印逻辑
│   │   └── colors.go      # 颜色方案
│   ├── forwarder/         # 请求转发逻辑
│   │   ├── forwarder.go   # 转发实现
│   │   └── client.go      # HTTP 客户端包装器
│   └── logger/            # 日志系统
│       ├── logger.go      # 日志实现
│       └── writer.go      # 日志写入器
├── pkg/request/           # 请求数据模型
│   └── request.go         # 请求结构定义
├── config.yaml.example    # 配置文件模板
├── Makefile              # 构建脚本
├── go.mod                # Go 模块定义
├── go.sum                # 依赖校验和
└── docs/                 # 文档
    └── README-zh.md      # 中文文档
```

## 输出示例

### 基本请求日志记录

```text
┌───────────────────────────── REQUEST #1 ───(2024-01-15T10:30:45+08:00)─┐
│                                                                     │
│  [GET] /api/users [FROM: 192.168.1.100]                           │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│ ─── Headers ───                                                     │
│                                                                     │
│   Accept: application/json                                          │
│   Authorization: [REDACTED]                                         │
│   User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64)           │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│ ─── Body (0 B) ───                                                  │
│                                                                     │
│   [Empty Body]                                                      │
│                                                                     │
└──────────────────────────────── END OF REQUEST #1 ─────────────────┘
```

### JSON 负载转发

```text
┌───────────────────────────── REQUEST #2 ───(2024-01-15T10:35:22+08:00)─┐
│                                                                     │
│  [POST] /webhook/github [FROM: 140.82.112.1]                       │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│ ─── Headers ───                                                     │
│                                                                     │
│   Content-Type: application/json                                    │
│   User-Agent: GitHub-Hookshot/abc123                               │
│   X-GitHub-Event: push                                             │
│   X-GitHub-Delivery: 12345678-1234-1234-1234-123456789012          │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│ ─── Body (1.2 kB) ───                                               │
│                                                                     │
│   {                                                                 │
│     "ref": "refs/heads/main",                                       │
│     "repository": {                                                 │
│       "name": "reqtap",                                             │
│       "full_name": "funnyzak/reqtap"                                │
│     },                                                              │
│     "pusher": {                                                     │
│       "name": "username",                                           │
│       "email": "user@example.com"                                   │
│     },                                                              │
│     "commits": [                                                    │
│       {                                                             │
│         "id": "abc123",                                             │
│         "message": "Update README",                                  │
│         "author": {                                                 │
│           "name": "Developer",                                      │
│           "email": "dev@example.com"                                │
│         }                                                           │
│       }                                                             │
│     ]                                                               │
│   }                                                                 │
│                                                                     │
└──────────────────────────────── END OF REQUEST #2 ─────────────────┘

→ Forwarding to http://localhost:3000/webhook... ✓
→ Forwarding to https://api.example.com/ingest... ✓
```

### 二进制内容检测

```text
┌───────────────────────────── REQUEST #3 ───(2024-01-15T10:40:15+08:00)─┐
│                                                                     │
│  [POST] /upload [FROM: 192.168.1.100]                              │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│ ─── Headers ───                                                     │
│                                                                     │
│   Content-Type: application/octet-stream                            │
│   Content-Length: 1024                                             │
│   User-Agent: curl/7.64.1                                           │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│ ─── Body (1 kB) ───                                                 │
│                                                                     │
│   [Binary Body: application/octet-stream, 1 kB. Content skipped.]   │
│                                                                     │
└──────────────────────────────── END OF REQUEST #3 ─────────────────┘
```

---

## 许可证

本项目采用 [MIT 许可证](LICENSE)。
