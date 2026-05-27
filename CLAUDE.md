# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Go CLI (`tg`) — a script-friendly, JSON-first Telegram client. Single module `github.com/vika2603/telegram-cli`, Go 1.25. User-facing command surface is in `README.md`.

## Build / test / lint

- Standard: `go build ./...`, `go test ./...`, `go vet ./...`.
- Format check is strict — CI fails on any `gofmt -l .` output. Fix with `gofmt -w .` or `goimports -local github.com/vika2603/telegram-cli -w .`.
- Lint: `golangci-lint run`. Version pinned to **v2.11.4** in CI — install the same locally via `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4`.
- `.golangci.yml` sets `goimports` local prefix `github.com/vika2603/telegram-cli`. Imports from this module must sit in their own group, separated from stdlib and third-party.
- Tests are unit-only with mocked Telegram sessions (`internal/telegram/session/fake.go`). Do **not** add tests that require real `api_id` / `api_hash` or live MTProto.

## Pre-commit checks

Before pushing, run all four locally and verify clean — CI runs the same set on every push, failing any one of them blocks the PR:

1. `gofmt -l .` (must be empty)
2. `go vet ./...`
3. `golangci-lint run` (v2.11.4, same as CI)
4. `go test ./...`

### `testifylint` gotcha

`require.Equal` on `float64` is rejected by golangci-lint (not visible to `go test` alone — only the lint step catches it). Use `require.InDelta(t, want, got, 0)` for exact compare or `require.InEpsilon` for relative tolerance. This bites JSON round-trip tests because `json.Unmarshal` into `map[string]any` decodes every number as `float64`.

## Error classification (important)

Error → exit code and JSON error code mapping is centralized in `internal/program/status`.
Adding a new user-visible error class requires updates in lockstep:

1. Sentinel in the package that owns the failure, e.g. `command.ErrUsage`, `config.ErrInvalid`, `account.ErrBusy`, `telegram/peer.ErrNotFound`, `telegram/message.ErrNoMedia`, or `telegram/session.ErrAuth`.
2. Case in `status.Code(err)` returning the stable JSON string code.
3. Case in `status.MapExitCode(err)` returning the exit code.

Existing codes (0, 1, 64–77, 130) are documented in `README.md` under "Exit codes" — keep that table in sync. Never call `os.Exit` from inside a command; return an error wrapping the owning sentinel (`fmt.Errorf("%w: ...", command.ErrUsage)`). The sole `os.Exit` lives in `cmd/tg/main.go`.

## Cobra command wiring

- `cmd/tg/main.go` is a thin shim; the real entry is `internal/program.Main()` so it stays testable.
- Every command package exposes `func New(inv *runtime.Invocation) *cobra.Command`. The invocation carries IO streams, config loader, account accessor, and version — no cobra types leak into runtime closures.
- Global flags (`--account`, `--config`, `--output`, `--color`, `--wait`, `--no-wait`, `--flood-wait-max`, `--quiet`) live on the root in `internal/cli/root/root.go`. Per-command flags go on the command itself; JSON/jq/template flags go through `output.AddJSONFlags`.
- Commands declare preconditions via `command.Meta` (`NeedsAccount`, `NeedsClient`, `SkipAuthCheck`) on `cmd.Annotations`. The root `PersistentPreRunE` reads Meta and short-circuits with `command.ErrPrecondition` / `session.ErrAuth` before the RunE runs — do not re-check these inside command bodies.
- Top-level commands must pick a `GroupID`: `"core"` or `"setup"`. New groups require an `AddGroup` call in `root.New`.
- `SilenceErrors` and `SilenceUsage` are both true on root — error rendering is owned by `output.EmitError`, invoked once in `program.Main`. Telegram session lifetime uses the callback form (`WithClient(ctx, acct, opts, fn)`) so bbolt and gotd resources tear down on every exit path.

## Output contract

- stdout is data-only. All diagnostics, prompts, and errors go to stderr.
- List commands emit ndjson (one JSON object per line); scalar commands emit a single object.
- Under `--json` / `--output=json`, errors on stderr are a single JSON object with `error`, `exit_code`, `message`.

## Runtime / config gotchas

- One `tg` process per account — enforced by a `flock` on `~/.config/tg/accounts/<name>/account.lock`. Concurrent runs return `account.ErrBusy` (exit **72**). Tests that touch real account dirs must use `t.TempDir()`.
- Config value precedence: **flag > env > config file > default**. Keep this order when adding new settings. Config struct fields are pointers so `nil` means "not set by this layer" during merge.
- `TG_LOG_FILE=""` (empty string) is meaningful — it resets log output to stderr. All other `TG_*` env vars treat empty as unset. Don't "normalize" this away.
- `TG_API_ID` and `TG_FLOOD_WAIT_MAX` must parse as integers; malformed values return `config.ErrInvalid` (exit 64), not a silent fallback.
- `api_hash` is a secret — never log it, never include it in error messages or JSON output. The `tg config show` command redacts it.

## Commits

Conventional Commits: `feat:`, `fix:`, `refactor:`, `chore:`, `docs:`, `test:`. Subject imperative, under ~72 chars.
