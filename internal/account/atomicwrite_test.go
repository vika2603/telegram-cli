package account

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofrs/flock"
	"github.com/stretchr/testify/require"
)

// testShortLockTimeout compresses the 5s production timeout to 200ms so the
// busy-lock assertion does not slow the test suite. The AtomicWrite timeout
// is a package-level `var` so tests can override it via t.Cleanup.
func init() { atomicWriteLockTimeout = 200 * time.Millisecond }

func TestAtomicWrite_createsFileAt0600(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "file.txt")
	lk := filepath.Join(dir, "account.lock")
	require.NoError(t, AtomicWrite(p, lk, []byte("hello")))
	data, err := os.ReadFile(p)
	require.NoError(t, err)
	require.Equal(t, "hello", string(data))
	info, err := os.Stat(p)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	// lock file should have been created 0600 too.
	linfo, err := os.Stat(lk)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), linfo.Mode().Perm())
}

func TestAtomicWrite_replacesExisting(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "file.txt")
	lk := filepath.Join(dir, "account.lock")
	require.NoError(t, os.WriteFile(p, []byte("old"), 0600))
	require.NoError(t, AtomicWrite(p, lk, []byte("new")))
	data, _ := os.ReadFile(p)
	require.Equal(t, "new", string(data))
}

func TestAtomicWrite_noLockAllowedForLockFileItself(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "account.lock")
	// Writing the lock file itself via lockPath="" is the documented escape.
	require.NoError(t, AtomicWrite(p, "", []byte{}))
	require.FileExists(t, p)
}

func TestAtomicWrite_busyLockWrapsTgerrErrBusy(t *testing.T) {
	dir := t.TempDir()
	lk := filepath.Join(dir, "account.lock")
	// Hold the lock in a goroutine so the main call hits the timeout path.
	require.NoError(t, os.WriteFile(lk, nil, 0600))
	holder := flock.New(lk)
	ok, err := holder.TryLock()
	require.NoError(t, err)
	require.True(t, ok)
	defer holder.Unlock()

	p := filepath.Join(dir, "file.txt")
	err = AtomicWrite(p, lk, []byte("x"))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrBusy, "busy lock must surface as ErrBusy for exit-72 mapping")
}
