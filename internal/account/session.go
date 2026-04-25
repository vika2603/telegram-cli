package account

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"

	"github.com/gotd/td/session"
)

// FileSessionStorage implements gotd's session.Storage. Reads are lock-free;
// writes acquire the account-level flock via AtomicWrite.
type FileSessionStorage struct {
	AccountName string
}

var _ session.Storage = (*FileSessionStorage)(nil)

// Layout: [magic(4)][version(2,BE)][len(4,BE)][payload].
const (
	sessionMagic    = "TGSN"
	sessionVersion  = uint16(1)
	sessionHeaderSz = 4 + 2 + 4
)

var (
	ErrSessionMagic   = errors.New("session.bin: bad magic (file not written by this binary)")
	ErrSessionVersion = errors.New("session.bin: unsupported version")
	ErrSessionLength  = errors.New("session.bin: declared length does not match file size")
)

func (s *FileSessionStorage) LoadSession(_ context.Context) ([]byte, error) {
	data, err := os.ReadFile(SessionFile(s.AccountName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, session.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(data) < sessionHeaderSz {
		return nil, fmt.Errorf("session.bin: short read (%d bytes, need >= %d): %w",
			len(data), sessionHeaderSz, ErrSessionMagic)
	}
	if string(data[:4]) != sessionMagic {
		return nil, ErrSessionMagic
	}
	version := binary.BigEndian.Uint16(data[4:6])
	if version != sessionVersion {
		return nil, fmt.Errorf("%w: file=%d, supported=%d",
			ErrSessionVersion, version, sessionVersion)
	}
	payloadLen := binary.BigEndian.Uint32(data[6:10])
	if int(payloadLen) != len(data)-sessionHeaderSz {
		return nil, fmt.Errorf("%w: declared=%d, actual=%d",
			ErrSessionLength, payloadLen, len(data)-sessionHeaderSz)
	}
	payload := make([]byte, payloadLen)
	copy(payload, data[sessionHeaderSz:])
	return payload, nil
}

func (s *FileSessionStorage) StoreSession(_ context.Context, data []byte) error {
	framed := make([]byte, sessionHeaderSz+len(data))
	copy(framed[:4], sessionMagic)
	binary.BigEndian.PutUint16(framed[4:6], sessionVersion)
	binary.BigEndian.PutUint32(framed[6:10], uint32(len(data)))
	copy(framed[sessionHeaderSz:], data)
	return AtomicWrite(SessionFile(s.AccountName), LockFile(s.AccountName), framed)
}

// DeleteSession removes the session file for name. Missing file is not an
// error (idempotent — supports multiple logout invocations on the same slot).
func DeleteSession(name string) error {
	if err := os.Remove(SessionFile(name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
