# Repository Guidelines

## Document Index
- `README.md`: Feature and installation overview; prioritize the Chinese version.
- `README-EN.md`: English quick start guide that can cross-reference this document.

## Project Structure & Module Organization
- `cmd/reqtap`: Cobra CLI and HTTP server entrypoint; start development here.
- `internal/`: Module-specific packages (`config`, `server`, `forwarder`, `web`, `printer`, `logger`, `static`); never create cross-directory circular dependencies.
- `pkg/request`: Reusable public request models; add new exported types here and keep go doc updated.
- `scripts/`: Installation and release scripts; when reusing a script, update the script index block as well.
- `build/`, `dist/`, `logs/`, `coverage/`: Build and runtime artifacts; keep them covered by `.gitignore`.

## Build, Test, and Development Commands
- `make deps`: Download dependencies and run `go mod tidy`; run after the first checkout or any Go version upgrade.
- `make build` / `go build -o build/reqtap ./cmd/reqtap`: Produce a binary for the local platform with version metadata.
- `make build-all`: Cross-compile batches of binaries into `dist/reqtap-<os>-<arch>`.
- `make test`: Run `go test -v ./...`; execute at least once before submitting changes.
- `make test-coverage`: Generate `coverage/coverage.html` to validate critical paths.
- `make lint`: Run `golangci-lint run`; install the tool locally if missing.

## Coding Style & Naming Conventions
- Format all Go code with gofmt (`make fmt`) and goimports; follow the tool output for tabs/spaces.
- Keep package and directory names short snake_case; export identifiers only when other packages need them and document with comments.
- Configuration keys stay in `snake_case`, environment variables use the `REQTAP_` prefix, and log fields keep lower_snake names such as `request_id` and `target_url`.
- Route any new bilingual CLI output through `internal/printer`; never call `fmt.Println` directly for user-facing messages.

## Testing Guidelines
- Default to Go testing plus `github.com/stretchr/testify/assert`; name test files `xxx_test.go` and functions `TestComponent_Scenario`.
- Target ≥80% coverage; mention coverage for new modules in the PR description and justify any gaps caused by IO or integration limits.
- For web console or forwarding regressions, prefer simulations offered by `internal/server`.
- After `make test-coverage`, attach the `coverage/coverage.html` screenshot or key metrics to the PR.

## Commit & Pull Request Guidelines
- Follow the `type: summary` format (for example `feat: add forwarder retries`, `chore:update deps`) and append `(#issue)` to match history.
- Keep each commit logically atomic: separate code, config, and documentation; exclude generated artifacts.
- Use the PR template: problem background, solution, verification method (logs/screenshots), regression risks, and include config diffs when relevant.
- Link the associated Issue or Release Milestone; synchronize README and this guide when touching CLI or configuration behavior.

## Security & Configuration Tips
- Store secrets in `.env` or environment variables—never in `config.yaml`; if an example is needed, change `config.yaml.example` instead.
- When debugging the web console locally, reset `web.admin_password`; never commit default weak credentials.
- Any new external endpoint must route through the `internal/forwarder` allowlist and retry strategy; avoid hardcoding URLs elsewhere.
