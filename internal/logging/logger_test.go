package logging

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/vika2603/telegram-cli/internal/config"
)

func newTestLogger(buf *bytes.Buffer) *zap.Logger {
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(buf), zapcore.DebugLevel)
	return zap.New(&redactingCore{Core: core}, zap.WithCaller(false))
}

func TestRedactingCore_masksKnownFields(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)
	log.Info("auth step",
		zap.String("api_hash", "SECRET_HASH_VALUE"),
		zap.String("code", "123456"),
		zap.String("password", "hunter2"),
		zap.String("session", "opaque-bytes"),
		zap.String("account", "alice"),
	)
	require.NoError(t, log.Sync())

	var obj map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &obj))
	require.Equal(t, "[redacted]", obj["api_hash"])
	require.Equal(t, "[redacted]", obj["code"])
	require.Equal(t, "[redacted]", obj["password"])
	require.Equal(t, "[redacted]", obj["session"])
	require.Equal(t, "alice", obj["account"])
	require.NotContains(t, buf.String(), "SECRET_HASH_VALUE")
}

func TestRedactingCore_maskingSurvivesWith(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf).With(zap.String("api_hash", "FROM_WITH"))
	log.Info("after with")
	require.NoError(t, log.Sync())

	var obj map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &obj))
	require.Equal(t, "[redacted]", obj["api_hash"])
	require.NotContains(t, buf.String(), "FROM_WITH")
}

func TestBuildLogger_closeCallbackClosesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tg.log")
	level := "info"
	format := "json"
	cfg := config.Config{Log: config.LogCfg{
		Level: &level, Format: &format, File: &path,
	}}
	probe := newClosingProbe(t)
	withCloseProbe(probe, func() {
		log, cleanup, err := BuildLogger(cfg)
		require.NoError(t, err)
		log.Info("hello from test", zap.String("k", "v"))
		require.Equal(t, 0, probe.count(), "file must be open before cleanup")
		cleanup()
		require.Equal(t, 1, probe.count(),
			"openedFile.Close must fire exactly once from cleanup")
		require.NotPanics(t, func() { cleanup() })
		require.Equal(t, 1, probe.count(),
			"second cleanup must be a no-op (idempotent)")
	})
	data, rerr := os.ReadFile(path)
	require.NoError(t, rerr)
	require.Contains(t, string(data), "hello from test")
}

type closingProbe struct {
	mu sync.Mutex
	n  int
}

func newClosingProbe(t *testing.T) *closingProbe {
	t.Helper()
	return &closingProbe{}
}
func (p *closingProbe) inc() {
	p.mu.Lock()
	p.n++
	p.mu.Unlock()
}
func (p *closingProbe) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.n
}

func withCloseProbe(p *closingProbe, fn func()) {
	saved := openFileForLog
	openFileForLog = func(path string, flag int, perm os.FileMode) (closableFile, error) {
		f, err := os.OpenFile(path, flag, perm)
		if err != nil {
			return nil, err
		}
		return &countingFile{File: f, probe: p}, nil
	}
	defer func() { openFileForLog = saved }()
	fn()
}

type countingFile struct {
	*os.File
	probe *closingProbe
}

func (c *countingFile) Close() error {
	c.probe.inc()
	return c.File.Close()
}

func TestBuildLogger_noFileNoLeak(t *testing.T) {
	level := "info"
	cfg := config.Config{Log: config.LogCfg{Level: &level}}
	log, cleanup, err := BuildLogger(cfg)
	require.NoError(t, err)
	require.NotNil(t, log)
	require.NotNil(t, cleanup)
	require.NotPanics(t, cleanup)
}

func TestBuildLogger_openFailureReturnsSafeCleanup(t *testing.T) {
	level := "info"
	bad := filepath.Join(t.TempDir(), "no-such-subdir", "tg.log")
	cfg := config.Config{Log: config.LogCfg{Level: &level, File: &bad}}
	log, cleanup, err := BuildLogger(cfg)
	require.Error(t, err)
	require.Nil(t, log)
	require.NotNil(t, cleanup)
	require.NotPanics(t, cleanup)
}
