package account

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// atomicWriteLockTimeout bounds how long we wait for account.lock before
// giving up with ErrBusy. `var` not `const` so tests can compress it via init().
var atomicWriteLockTimeout = 5 * time.Second

// AtomicWrite writes data to path via tmp + rename. Creates the parent
// directory with mode 0700 if missing. File mode is 0600.
//
// Any write to account.json or session.bin MUST first acquire an exclusive
// flock on account.lock; pass lockPath="" to skip locking, which is ONLY
// legitimate for writing account.lock itself. A timeout wraps
// ErrBusy so callers can map it to the busy exit code.
func AtomicWrite(path, lockPath string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	if lockPath != "" {
		// Ensure the lock file exists before flock wraps it — flock only
		// opens O_RDONLY and does not create.
		if f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY, 0600); err != nil {
			return fmt.Errorf("touch lock %s: %w", lockPath, err)
		} else {
			_ = f.Close()
		}
		lk := flock.New(lockPath)
		deadline := time.Now().Add(atomicWriteLockTimeout)
		for {
			ok, err := lk.TryLock()
			if err != nil {
				return fmt.Errorf("flock %s: %w", lockPath, err)
			}
			if ok {
				defer func() { _ = lk.Unlock() }()
				break
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("flock %s: account.lock busy after %s: %w", lockPath, atomicWriteLockTimeout, ErrBusy)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create tmp in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
