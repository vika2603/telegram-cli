package account

import (
	"os"
	"path/filepath"
)

// Root returns the root tg config directory, honoring XDG_CONFIG_HOME.
// Default: $HOME/.config/tg.
func Root() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "tg")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tg")
}

func ConfigFile() string { return filepath.Join(Root(), "config.toml") }

func AccountsDir() string { return filepath.Join(Root(), "accounts") }

// AccountDir returns the directory for account <name>. Does not validate name;
// callers must validate via IsValidName (see meta.go) first.
func AccountDir(name string) string {
	return filepath.Join(AccountsDir(), name)
}

func MetaFile(name string) string    { return filepath.Join(AccountDir(name), "account.json") }
func SessionFile(name string) string { return filepath.Join(AccountDir(name), "session.bin") }
func LockFile(name string) string    { return filepath.Join(AccountDir(name), "account.lock") }
func PeersDB(name string) string     { return filepath.Join(AccountDir(name), "peers.db") }
