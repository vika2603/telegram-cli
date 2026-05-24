// Package daemon manages per-account background services that hold a
// long-lived MTProto session. Each authed account name gets one daemon
// instance owned by the host's native service manager (launchd on macOS,
// systemd-user on Linux, schtasks on Windows). All on-disk artifacts —
// service definition file, log, update stream, Phase 3 IPC socket — live
// under the same per-account directory as the existing account.lock and
// bbolt session storage, so the directory remains the single source of
// truth for one account's runtime state.
package daemon

import (
	"path/filepath"

	"github.com/vika2603/telegram-cli/internal/account"
)

// DaemonDir is the per-account directory holding all daemon-owned files.
// It sits inside the existing account directory so removing an account
// also removes its daemon assets.
func DaemonDir(accountName string) string {
	return filepath.Join(account.AccountDir(accountName), "daemon")
}

// MetaFile is the JSON sidecar carrying log path, install time, etc.
// Used by status/logs subcommands to locate artifacts without parsing
// service definitions.
func MetaFile(accountName string) string {
	return filepath.Join(DaemonDir(accountName), "daemon.json")
}

// LogFile is the daemon's stdout/stderr target. Service definitions
// pin both streams to this path; logrotate trims it in process.
func LogFile(accountName string) string {
	return filepath.Join(DaemonDir(accountName), "daemon.log")
}

// UpdatesFile is the Phase-2 ndjson stream of MTProto updates the worker
// appends to. Once Phase 3 lands a Unix socket, the worker continues to
// tee here so consumers can either tail the file or subscribe over IPC.
func UpdatesFile(accountName string) string {
	return filepath.Join(DaemonDir(accountName), "updates.ndjson")
}

// SocketPath is the planned Phase-3 Unix socket path. Declared here so
// the Manager can advertise it via Meta even before the IPC server is
// implemented.
func SocketPath(accountName string) string {
	return filepath.Join(DaemonDir(accountName), "daemon.sock")
}
