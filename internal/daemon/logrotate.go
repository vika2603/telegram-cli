package daemon

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// RotateIfLarger renames path to path+".1" if its current size exceeds
// maxBytes. It is intentionally simple: a single backup is kept, prior
// path+".1" is overwritten. This matches cc-connect's behavior and is
// sufficient for the per-account, low-volume daemon log we produce.
//
// maxBytes <= 0 disables rotation. Missing file is a no-op.
func RotateIfLarger(path string, maxBytes int64) error {
	if maxBytes <= 0 {
		return nil
	}
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Size() <= maxBytes {
		return nil
	}
	backup := path + ".1"
	// Best-effort: remove a previous backup before renaming so a
	// missing rename target doesn't leave the rotated file orphaned.
	_ = os.Remove(backup)
	if err := os.Rename(path, backup); err != nil {
		return err
	}
	// Touch the new file so subsequent O_APPEND opens are immediate
	// (service definitions typically rely on append:path semantics).
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	return f.Close()
}

// EnsureDir creates dir with mode 0o700 if missing. A thin helper used
// by both the worker and the platform managers so log/updates files
// always have a parent before first write.
func EnsureDir(dir string) error {
	return os.MkdirAll(filepath.Clean(dir), 0o700)
}
