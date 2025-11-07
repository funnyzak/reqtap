# Repository Guidelines

## 文档索引
- `README.md`：功能与安装概览，中文版优先。
- `README-EN.md`：英文快速入门，可与本指南交叉引用。

## Project Structure & Module Organization
- `cmd/reqtap`：Cobra CLI 与 HTTP 服务器入口，开发时从这里启动。
- `internal/`：按模块拆分（`config`, `server`, `forwarder`, `web`, `printer`, `logger`, `static`），禁止跨目录循环依赖。
- `pkg/request`：对外可复用的请求模型，新增公共类型应放入此处并保持 go doc。
- `scripts/`：安装与发布脚本，复用时请同步更新脚本索引块。
- `build/`, `dist/`, `logs/`, `coverage/`：构建与运行产物，保持在 `.gitignore` 控制范围。

## Build, Test, and Development Commands
- `make deps`：下载依赖并执行 `go mod tidy`；首次拉取或升级 Go 版本后必跑。
- `make build` / `go build -o build/reqtap ./cmd/reqtap`：为本机平台产出带版本信息的二进制。
- `make build-all`：批量交叉编译，生成 `dist/reqtap-<os>-<arch>` 文件。
- `make test`：运行 `go test -v ./...`；提交前至少一次。
- `make test-coverage`：产出 `coverage/coverage.html`，用于验证关键路径。
- `make lint`：运行 `golangci-lint run`，若未安装会提示，请本地安装保持一致。

## Coding Style & Naming Conventions
- 全部 Go 代码使用 gofmt（`make fmt`）与 goimports，四空格/Tab 以工具输出为准。
- 包与目录使用短小的蛇形小写；导出符号只在需要被其他包调用时首字母大写并添加注释。
- 配置键保持 `snake_case`，环境变量使用 `REQTAP_` 前缀；日志字段沿用 `request_id`, `target_url` 等 lower_snake。
- 新增中英文 CLI 输出需通过 `internal/printer` 的集中渲染，禁止直接 `fmt.Println`。

## Testing Guidelines
- 默认使用 Go testing + `github.com/stretchr/testify/assert`；测试文件命名为 `xxx_test.go`，函数采用 `TestComponent_Scenario`。
- 覆盖率目标≥80%，新增模块需在 PR 描述中说明覆盖率；如因 IO/接入受限，应给出跳过原因。
- 对 Web 控制台或转发逻辑的回归测试，优先使用 `internal/server` 提供的模拟请求。
- `make test-coverage` 后请附 `coverage/coverage.html` 截图或核心指标到 PR。

## Commit & Pull Request Guidelines
- 遵循 "type: summary"（如 `feat: add forwarder retries`、`chore:update deps`）并在尾部附 `(#issue)`，与现有历史保持一致。
- 单次提交保持逻辑原子：代码、配置、文档分别分 commit，禁止混入生成文件。
- PR 描述模板：问题背景、解决方案、验证方式（命令输出或截图）、回归风险，必要时附配置 diff。
- 关联 Issue 或 Release Milestone，涉及 CLI/配置变更需同步更新 README 与本文件相关段落。

## Security & Configuration Tips
- 机密键放入 `.env` 或环境变量，不要写入 `config.yaml`；示例请改动 `config.yaml.example`。
- 本地调试 Web 控制台时确认 `web.admin_password` 已重置，自带弱口令不得提交。
- 任何引入的新外部端点必须通过 `internal/forwarder` 的 allowlist 与重试策略，避免硬编码 URL。
