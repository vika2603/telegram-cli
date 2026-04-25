// Package accounttest provides helpers that create temporary tg config
// roots and seed account state for tests. Production code must never
// import this package.
package accounttest

import (
	"testing"

	"github.com/vika2603/telegram-cli/internal/account"
)

// TempConfigRoot redirects XDG_CONFIG_HOME to t.TempDir(), clears all
// TG_* environment variables, and returns the root path (the directory
// XDG_CONFIG_HOME points at; tg's tree lives under root+"/tg").
func TempConfigRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	// Clear any inherited TG_* overrides that would break precedence.
	for _, k := range []string{
		"TG_ACCOUNT", "TG_API_ID", "TG_API_HASH", "TG_OUTPUT", "TG_COLOR",
		"TG_LOG_LEVEL", "TG_LOG_FILE", "TG_LOG_FORMAT", "TG_FLOOD_WAIT_MODE",
		"TG_FLOOD_WAIT_MAX", "TG_CONFIG",
	} {
		t.Setenv(k, "")
	}
	return dir
}

// SeedAccount writes a valid account.json under <root>/tg/accounts/<name>/
// with the given state. Returns the *account.Account that account.LoadAccount
// would return. Use alongside TempConfigRoot.
func SeedAccount(t *testing.T, root, name string, state account.State) *account.Account {
	t.Helper()
	_ = root

	meta := account.Meta{
		Name:  name,
		State: state,
	}
	// AddAccount persists account.json; if state is empty it defaults to StateNEW.
	if err := account.AddAccount(meta); err != nil {
		t.Fatalf("SeedAccount AddAccount: %v", err)
	}
	// Override the state if the caller asked for something other than the
	// default (StateNEW) that AddAccount writes when State == "".
	if state != "" && state != account.StateNEW {
		if err := account.SetState(name, state); err != nil {
			t.Fatalf("SeedAccount SetState: %v", err)
		}
	}
	acct, err := account.LoadAccount(name)
	if err != nil {
		t.Fatalf("SeedAccount LoadAccount: %v", err)
	}
	return acct
}
