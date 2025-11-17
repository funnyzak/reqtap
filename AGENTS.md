# Repository Guidelines

## Project Structure & Module Organization
`cmd/reqtap/main.go` bootstraps the CLI, while request capture, forwarding, and web console code lives in `internal/` (`server`, `web`, `forwarder`, `static`, `printer`, `logger`). Reusable DTOs and helpers that may be imported elsewhere belong in `pkg/request`. Generated assets stay in `build/` or `dist/`, and configuration samples sit at `config.yaml.example`; installer logic is under `scripts/install.sh`.

## Build, Test, and Development Commands
- `make build` – compile the platform binary into `build/reqtap`.
- `make dev` – run the hot-reload workflow (requires `air`, falls back to `go run`).
- `make check` – sequential `fmt`, `golangci-lint run`, and `go test`; gate every PR with it.
- `make test-coverage` – emit `coverage/coverage.html` for quick diff-able reports.
- `make docker-build && make docker-run` – validate the Docker image on `:38888`.

## Coding Style & Naming Conventions
Keep Go sources `gofmt`-clean (tabs indentation, grouped imports). Use descriptive lowercase package names and PascalCase for exported types/functions. Favor short receiver names, wrap errors with context, and inject dependencies via constructors. Configuration defaults should be centralized in YAML or env lookups instead of literals.

## Testing Guidelines
Store unit tests next to the code they cover (`*_test.go`, e.g., `internal/config/config_test.go`). Follow `TestXxx` and table-driven patterns for request parsing, exporter filters, and forwarder retries. Run `go test -v ./...` (or `make test`) before pushes, and refresh coverage when touching concurrency or logging paths. Add regression tests when redaction, export formats, or websocket behavior changes.

## Commit & Pull Request Guidelines
Follow the existing history by using `<type>: <imperative summary>` (e.g., `docs: readme`) or a concise sentence plus PR reference (`Enhance export formats (#6)`). Keep commits focused, reference the related issue, and paste `make check` output plus any new screenshots for web assets. PR descriptions must note config or API impacts so downstream installers stay aligned.

## Security & Configuration Tips
Never embed credentials; point contributors to env vars or `config.yaml`. Override the sample `web.auth.users` accounts before demos, sanitize files under `logs/` before sharing, and rotate any `forward.urls` tokens after tests. Release artifacts pick up the same environment variables, so document required keys in README updates.

## Documentation & Support Map
Keep this file aligned with `README.md`, `README-EN.md`, and `config.yaml.example`. When new modules or scripts appear, summarize them in the READMEs and mention them here so the documentation index stays accurate.
