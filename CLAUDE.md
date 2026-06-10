# CLAUDE.md

## Project Overview

Redis TUI Manager — a terminal user interface for managing Redis databases, built with Go and Bubble Tea. Alternative to redis-cli with interactive browsing, editing, monitoring, and advanced operations.

## Build & Test Commands

```bash
make build        # Build binary to bin/redis-tui
make test         # Run tests: go test -v ./...
make test-cover   # Tests with coverage report
make lint         # Run go vet
make fmt          # Format code with go fmt
make run          # Run the application
make dev-deps         # Install goreleaser
make test-cover-check # Tests + fail if any function < 100% coverage
```

CI runs `go test -v -race -coverprofile=coverage.out ./...` — always ensure tests pass with `-race`. CI enforces **100% statement coverage** per function — any new code must include tests.

## Architecture

```
main.go                    # Entry point, CLI flag parsing, config init
internal/
  cmd/                     # Bubble Tea command factories (return tea.Cmd)
  ui/                      # Bubble Tea UI (Model/Update/View pattern)
  redis/                   # Redis client wrapper (standalone + cluster)
  types/                   # Shared type definitions and messages
  db/                      # Config persistence (~/.config/redis-tui/config.json)
  service/                 # Interfaces (ConfigService, RedisService) and DI container
  testutil/                # Test helpers and mock implementations
```

**Bubble Tea message flow**: KeyMsg → handleKeyPress() → Command (tea.Cmd) → async execution → Message (tea.Msg) → Update() → View()

## Code Conventions

- **Go version**: 1.26.4 (set in go.mod)
- **Package names**: lowercase, single word (`cmd`, `redis`, `types`, `db`, `ui`)
- **Receivers**: short names (`c *Client`, `m Model`, `cfg *Config`)
- **Message types**: PascalCase with `Msg` suffix (`ConnectedMsg`, `KeysLoadedMsg`)
- **Command methods**: return `tea.Cmd` — pattern: `func (c *Commands) Op() tea.Cmd { return func() tea.Msg { ... } }`
- **Error handling**: return `(result, error)` tuples, wrap with `fmt.Errorf("context: %w", err)`
- **Logging**: `log/slog` structured logging (`slog.Error()`, `slog.Warn()`, `slog.Info()`)
- **Dependency injection**: services injected via `Commands{config, redis}` struct; use interfaces from `service/interfaces.go`
- **Thread safety**: `sync.RWMutex` for shared state (see `db/config.go`)

## Testing

- Tests use Go standard `testing` package
- Redis tests use `alicebob/miniredis` for in-memory Redis
- Mock implementations in `internal/testutil/` (`mock_redis.go`, `mock_redis_full.go`)
- Generic assertion helpers: `AssertEqual[T]`, `AssertNoError`, `AssertError`
- Test files live alongside source files (`*_test.go`)

### Test rules

- **Never suppress errors** — no `_, _ :=` or bare calls that return errors. Every error must be checked, even in test setup code. Use `t.Fatalf("... failed: %v", err)` for setup errors.
- **Config persistence round-trip** — any new config feature must have a test that writes data, reloads from disk via `reloadConfig(t, cfg)`, and asserts every field survived the JSON round-trip. This catches broken JSON tags.
- **Mocks simulate, never re-implement** — mock implementations should return configured values/errors. Never duplicate real business logic in mocks.
- **Test files are exempt from file length limits** — keep all tests for a given source file in a single `*_test.go` file alongside the source (e.g., `operations.go` ↔ `operations_test.go`). Do not split tests across `_extra_test.go` or similarly suffixed files just to stay under a length threshold.

### Test helpers (`internal/db/config_test.go`)

- `newTestConfig(t)` — creates a temp-dir-backed config
- `reloadConfig(t, cfg)` — creates a fresh `NewConfig` from the same file path, forcing a full JSON save/load cycle

## Key Dependencies

- `charm.land/bubbletea/v2` — TUI framework
- `charm.land/bubbles/v2` — TUI components
- `charm.land/lipgloss/v2` — Terminal styling
- `redis/go-redis/v9` — Redis client
- `alicebob/miniredis/v2` — In-memory Redis for tests
- `alecthomas/chroma/v2` — Syntax highlighting

## Git Conventions

- **Conventional commits**: `feat:`, `fix:`, `docs:`, `test:`, `perf:`, `refactor:`, `chore:`
- Keep subject line under 50 characters, imperative mood
- **Net new changes ship via PR** — never push directly to `main`. After committing, create a feature branch and open a pull request using the `/pull-request` skill. If the change resolves a tracked issue, the PR description must back-link it (e.g. `Closes #29`) so GitHub auto-closes on merge.
- **Standing authorization for PR plumbing**: pushing to a feature branch (`git push -u origin <branch>`) and opening the PR (`gh pr create --base main --body-file …`) do not need per-PR confirmation once the body has been shown and approved. Force-pushing, pushing to `main`, merging, or closing PRs still require explicit approval.

## Guardrails

- All Redis operations must go through `internal/redis/` — never use `go-redis` directly elsewhere
- Config schema changes in `internal/db/` must be backward-compatible with existing `~/.config/redis-tui/config.json` files
- All new command methods must go through the `Commands` struct with injected services — no global state
- New message types must follow the `Msg` suffix convention and be defined in the appropriate `messages_*.go` file
- **JSON tags on `types.Connection`** — changing JSON tags (especially to `json:"-"`) will break persistence. Round-trip tests in `config_test.go` enforce this.
- **Password stripping** — `save()` in `config.go` intentionally strips `Connection.Password`, `SSHConfig.Password`, and `SSHConfig.Passphrase` before writing to disk. This is a security invariant tested by `TestConfig_Persistence_PasswordStripping` and `TestConfig_Persistence_SSHPasswordStripping`. Never bypass this.
- **`UpdateConnection` field preservation** — `UpdateConnection()` preserves `Group`, `Color`, `UseSSH`, `SSHConfig`, `UseTLS`, `TLSConfig` from the existing connection. This is tested and must not regress.
- **SSH tunneling** — `UseSSH`, `SSHConfig`, `ScreenSSHTunnel`, and `SSHTunnelConnectedMsg` are defined and persisted but not yet implemented (no SSH client code exists). These fields are intentionally kept for future implementation. Do not remove them.

## Release

- GoReleaser v2.13.1 builds for Linux/macOS/Windows (amd64/arm64)
- Homebrew tap: `davidbudnick/homebrew-tap`
- Version injected via ldflags (`-X main.version=...`)
- CGO disabled (`CGO_ENABLED=0`)
