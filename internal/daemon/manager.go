package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DefaultLogMaxSize is the per-account log rotation threshold (10 MB).
// Smaller than cc-connect's default because there is only one account
// worth of traffic per file.
const DefaultLogMaxSize = 10 * 1024 * 1024

// Config is the static configuration baked into a service definition at
// install time. Field semantics mirror cc-connect's daemon.Config so the
// install/uninstall flow is familiar to anyone who has used that
// project, with one addition (Account) — each daemon is scoped to a
// single account name, since tg's flock contract is per-account.
type Config struct {
	// Account is the slot name passed to "tg --account ... daemon run".
	// Required. Used to derive ServiceLabel, paths, and the worker's
	// account selection.
	Account string

	// BinaryPath is the absolute path to the tg executable the service
	// will launch. Defaults to the running binary's resolved symlink
	// target at install time.
	BinaryPath string

	// LogFile receives stdout+stderr of the worker. Defaults to
	// LogFile(Account) under the account's daemon dir.
	LogFile string

	// LogMaxSize is the in-process rotation threshold in bytes.
	// Defaults to DefaultLogMaxSize.
	LogMaxSize int64

	// EnvPATH captures the user's shell PATH at install time so the
	// service can locate auxiliary executables (gh, etc.). Services
	// inherit minimal env from launchd/systemd; without this, lookups
	// commonly fail when the daemon runs.
	EnvPATH string

	// EnvExtra captures proxy variables (HTTP_PROXY, HTTPS_PROXY,
	// NO_PROXY and their lowercase variants) so MTProto traffic can
	// traverse a corporate proxy when the user is behind one.
	EnvExtra map[string]string
}

// Status is the platform-agnostic view of an installed (or
// uninstalled) daemon service. Manager.Status fills it from the host's
// service manager.
type Status struct {
	Installed bool
	Running   bool
	PID       int
	Platform  string // "launchd", "systemd (user)", "schtasks", "unsupported"
	Account   string // the slot this status refers to
}

// Manager is the per-account interface every platform implements. It is
// intentionally narrow: install/start/stop/restart/uninstall/status,
// nothing else. Anything richer (subscribe, query, send) belongs in
// Phase 3 over the Unix socket.
type Manager interface {
	Install(cfg Config) error
	Uninstall() error
	Start() error
	Stop() error
	Restart() error
	Status() (*Status, error)
	Platform() string
}

// NewManager dispatches to the host's native service manager. The
// returned Manager is bound to accountName for the lifetime of the
// process; switching accounts requires a fresh NewManager call.
func NewManager(accountName string) (Manager, error) {
	if accountName == "" {
		return nil, errors.New("daemon manager requires non-empty account name")
	}
	return newPlatformManager(accountName)
}

// Meta is the JSON sidecar that survives across "tg daemon" invocations.
// status/logs read this to find the log file path without parsing the
// service definition.
type Meta struct {
	Account     string `json:"account"`
	LogFile     string `json:"log_file"`
	LogMaxSize  int64  `json:"log_max_size"`
	BinaryPath  string `json:"binary_path"`
	InstalledAt string `json:"installed_at"`
	Platform    string `json:"platform"`
	// SocketPath is reserved for Phase 3 (Unix socket IPC). Recording
	// the planned path now lets us roll it out without bumping the
	// Meta schema version.
	SocketPath string `json:"socket_path,omitempty"`
}

// SaveMeta writes m to the per-account meta file, creating directories
// as needed. The write is best-effort — failure to persist meta does
// not invalidate an otherwise-successful install.
func SaveMeta(m *Meta) error {
	if m.Account == "" {
		return errors.New("save meta: missing account")
	}
	path := MetaFile(m.Account)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create daemon dir: %w", err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// LoadMeta reads the meta sidecar for accountName. Returns os.ErrNotExist
// when the daemon has never been installed for this account.
func LoadMeta(accountName string) (*Meta, error) {
	data, err := os.ReadFile(MetaFile(accountName))
	if err != nil {
		return nil, err
	}
	var m Meta
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse meta: %w", err)
	}
	return &m, nil
}

// RemoveMeta deletes the meta sidecar. Used by Uninstall. Silently
// succeeds on ENOENT — uninstall is idempotent.
func RemoveMeta(accountName string) error {
	err := os.Remove(MetaFile(accountName))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// NowISO is the canonical timestamp format for Meta.InstalledAt. Kept
// as a package-level helper so tests can shadow it for determinism.
var NowISO = func() string { return time.Now().UTC().Format(time.RFC3339) }

// Resolve fills the zero-valued fields of cfg with defaults derived
// from the runtime environment: binary path from os.Executable, log
// path from LogFile(account), env from os.Getenv. Callers populate
// Account before calling; any other field they leave zero gets a
// sensible default here.
func Resolve(cfg *Config) error {
	if cfg.Account == "" {
		return errors.New("daemon config requires an account name")
	}
	if cfg.BinaryPath == "" {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("detect binary path: %w", err)
		}
		if resolved, rerr := filepath.EvalSymlinks(exe); rerr == nil {
			exe = resolved
		}
		cfg.BinaryPath = exe
	}
	if cfg.LogFile == "" {
		cfg.LogFile = LogFile(cfg.Account)
	}
	if cfg.LogMaxSize <= 0 {
		cfg.LogMaxSize = DefaultLogMaxSize
	}
	if cfg.EnvPATH == "" {
		cfg.EnvPATH = os.Getenv("PATH")
	}
	if cfg.EnvExtra == nil {
		cfg.EnvExtra = captureDaemonEnv()
	}
	return nil
}

// captureDaemonEnv snapshots proxy variables at install time. Service
// processes have empty env on most platforms, so anything the worker
// needs to find (proxies, custom DNS resolvers wired via env, etc.)
// must be propagated explicitly.
func captureDaemonEnv() map[string]string {
	keys := []string{
		"http_proxy", "https_proxy", "no_proxy",
		"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY",
		"all_proxy", "ALL_PROXY",
	}
	env := map[string]string{}
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			env[k] = v
		}
	}
	return env
}
