package account

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"time"
)

// ErrAccountExists is returned by AddAccount when the target name already has
// an on-disk directory. Callers in the cli layer wrap it as command.ErrUsage so
// a duplicate-add surfaces as exit 64 rather than the default 1.
var ErrAccountExists = errors.New("account already exists")

// ErrAccountNotFound is returned by ReadMeta when account.json is missing —
// distinct from a genuine IO / parse error on a present file. Callers use
// errors.Is to translate it into a user-friendly "account <name> does not
// exist" message rather than the raw "open ...: no such file or directory".
var ErrAccountNotFound = errors.New("account not found")

// The default account lives in config.toml under `default_account` and is
// written by `tg account use`. This package does not own that persistence;
// ResolveAccount takes the already-loaded config value so resolution is a
// pure function of caller inputs.

func ListAccounts() ([]string, error) {
	dir := AccountsDir()
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && IsValidName(e.Name()) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func AddAccount(m Meta) error {
	if !IsValidName(m.Name) {
		return fmt.Errorf("invalid account name %q", m.Name)
	}
	dir := AccountDir(m.Name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("%w: %s", ErrAccountExists, m.Name)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if m.CreatedAt == 0 {
		m.CreatedAt = time.Now().Unix()
	}
	if m.State == "" {
		m.State = StateNEW
	}
	if err := WriteMeta(m); err != nil {
		_ = os.RemoveAll(dir)
		return err
	}
	return nil
}

// RemoveAccount removes the entire account directory for name. The caller
// MUST have released the account flock (LockFile(name)) before calling
// this; otherwise the directory will race with an in-flight process. A
// missing directory is not an error (idempotent).
//
// This does not update config.toml — callers that want to clear a
// default_account pointer must call config.UnsetDefaultAccount themselves
// AFTER RemoveAccount returns.
func RemoveAccount(name string) error {
	if err := os.RemoveAll(AccountDir(name)); err != nil {
		return fmt.Errorf("remove account dir: %w", err)
	}
	return nil
}

// ResolveAccount returns the explicit name if non-empty; otherwise falls back
// to configDefault (the value of `default_account` from the merged config).
// Does not validate existence; callers read meta to verify.
func ResolveAccount(explicit, configDefault string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if configDefault != "" {
		return configDefault, nil
	}
	return "", errors.New("no account specified and no default set (set via `tg account use <name>` or pass --account)")
}

// SetState updates the State field in account.json for the given name.
func SetState(name string, state State) error {
	m, err := ReadMeta(name)
	if err != nil {
		return err
	}
	m.State = state
	return WriteMeta(m)
}
