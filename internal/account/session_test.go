package account

import (
	"context"
	"encoding/binary"
	"os"
	"testing"

	"github.com/gotd/td/session"
	"github.com/stretchr/testify/require"
)

func TestFileSessionStorage_roundtrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(AccountDir("alice"), 0700))
	s := &FileSessionStorage{AccountName: "alice"}
	require.NoError(t, s.StoreSession(context.Background(), []byte("opaque")))
	got, err := s.LoadSession(context.Background())
	require.NoError(t, err)
	require.Equal(t, []byte("opaque"), got)
}

func TestFileSessionStorage_loadMissing_returnsErrNotFound(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(AccountDir("alice"), 0700))
	s := &FileSessionStorage{AccountName: "alice"}
	_, err := s.LoadSession(context.Background())
	require.ErrorIs(t, err, session.ErrNotFound)
}

func TestFileSessionStorage_writeIsAtomicUnderLock(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(AccountDir("alice"), 0700))
	s := &FileSessionStorage{AccountName: "alice"}
	require.NoError(t, s.StoreSession(context.Background(), []byte("first")))
	require.NoError(t, s.StoreSession(context.Background(), []byte("second")))
	got, err := s.LoadSession(context.Background())
	require.NoError(t, err)
	require.Equal(t, []byte("second"), got)
	_, err = os.Stat(SessionFile("alice") + ".tmp")
	require.True(t, os.IsNotExist(err))
}

func TestFileSessionStorage_storeWritesWrapper(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(AccountDir("alice"), 0700))
	s := &FileSessionStorage{AccountName: "alice"}
	payload := []byte("payload-bytes")
	require.NoError(t, s.StoreSession(context.Background(), payload))

	raw, err := os.ReadFile(SessionFile("alice"))
	require.NoError(t, err)
	require.Len(t, raw, sessionHeaderSz+len(payload))
	require.Equal(t, sessionMagic, string(raw[:4]))
	require.Equal(t, sessionVersion, binary.BigEndian.Uint16(raw[4:6]))
	require.Equal(t, uint32(len(payload)), binary.BigEndian.Uint32(raw[6:10]))
	require.Equal(t, payload, raw[sessionHeaderSz:])
}

func TestFileSessionStorage_loadRejectsBadMagic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(AccountDir("alice"), 0700))
	bogus := make([]byte, sessionHeaderSz+4)
	copy(bogus[:4], "XXXX")
	binary.BigEndian.PutUint16(bogus[4:6], sessionVersion)
	binary.BigEndian.PutUint32(bogus[6:10], 4)
	require.NoError(t, os.WriteFile(SessionFile("alice"), bogus, 0600))
	s := &FileSessionStorage{AccountName: "alice"}
	_, err := s.LoadSession(context.Background())
	require.ErrorIs(t, err, ErrSessionMagic)
}

func TestFileSessionStorage_loadRejectsWrongVersion(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(AccountDir("alice"), 0700))
	bogus := make([]byte, sessionHeaderSz+4)
	copy(bogus[:4], sessionMagic)
	binary.BigEndian.PutUint16(bogus[4:6], 99)
	binary.BigEndian.PutUint32(bogus[6:10], 4)
	require.NoError(t, os.WriteFile(SessionFile("alice"), bogus, 0600))
	s := &FileSessionStorage{AccountName: "alice"}
	_, err := s.LoadSession(context.Background())
	require.ErrorIs(t, err, ErrSessionVersion)
}

func TestDeleteSession_MissingIsNoError(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// Directory does not exist; SessionFile points to a non-existent path.
	err := DeleteSession("ghost")
	require.NoError(t, err)

	// Create the dir and a session file, then delete it twice.
	require.NoError(t, os.MkdirAll(AccountDir("bob"), 0700))
	require.NoError(t, os.WriteFile(SessionFile("bob"), []byte("data"), 0600))
	require.NoError(t, DeleteSession("bob"))
	// Second call: file is already gone.
	require.NoError(t, DeleteSession("bob"))
}

func TestFileSessionStorage_loadRejectsLengthMismatch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(AccountDir("alice"), 0700))
	bogus := make([]byte, sessionHeaderSz+4)
	copy(bogus[:4], sessionMagic)
	binary.BigEndian.PutUint16(bogus[4:6], sessionVersion)
	binary.BigEndian.PutUint32(bogus[6:10], 99)
	require.NoError(t, os.WriteFile(SessionFile("alice"), bogus, 0600))
	s := &FileSessionStorage{AccountName: "alice"}
	_, err := s.LoadSession(context.Background())
	require.ErrorIs(t, err, ErrSessionLength)
}
