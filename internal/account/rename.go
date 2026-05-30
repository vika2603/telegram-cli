package account

import (
	"fmt"
	"os"

	"github.com/gofrs/flock"
)

// RenameAccount renames a local account slot from old to newName: it moves the
// account directory and rewrites account.json's Name field. It refuses when
// newName is invalid, old does not exist, newName already exists, or the
// account is currently in use (its flock is held by another process).
//
// It does NOT touch config.toml's default_account or any host daemon
// registration — those belong to higher layers (see action/auth.Rename), since
// the daemon's service name is bound to the account name and renaming under it
// would orphan the service.
func RenameAccount(old, newName string) error {
	if !IsValidName(newName) {
		return fmt.Errorf("invalid account name %q", newName)
	}
	if old == newName {
		return fmt.Errorf("%w: %s", ErrAccountExists, newName)
	}
	meta, err := ReadMeta(old)
	if err != nil {
		return err
	}
	if _, err := os.Stat(AccountDir(newName)); err == nil {
		return fmt.Errorf("%w: %s", ErrAccountExists, newName)
	}
	if err := ensureNotBusy(old); err != nil {
		return err
	}
	if err := os.Rename(AccountDir(old), AccountDir(newName)); err != nil {
		return fmt.Errorf("rename account dir: %w", err)
	}
	meta.Name = newName
	if err := WriteMeta(meta); err != nil {
		// Roll the directory back so a failed rewrite doesn't strand the slot
		// under the new name with a stale Name field.
		_ = os.Rename(AccountDir(newName), AccountDir(old))
		return fmt.Errorf("rewrite account meta: %w", err)
	}
	return nil
}

// ensureNotBusy returns ErrBusy when another process holds the account's
// flock. It only probes (lock then release immediately); the caller renames
// right after, accepting a tiny window — renaming a slot that is in active use
// is exactly the misuse this guards against.
func ensureNotBusy(name string) error {
	lockPath := LockFile(name)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("touch lock %s: %w", lockPath, err)
	}
	_ = f.Close()
	lk := flock.New(lockPath)
	ok, err := lk.TryLock()
	if err != nil {
		return fmt.Errorf("flock %s: %w", lockPath, err)
	}
	if !ok {
		return fmt.Errorf("%w: account %q is in use", ErrBusy, name)
	}
	_ = lk.Unlock()
	return nil
}
