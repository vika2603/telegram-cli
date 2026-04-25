package command

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/vika2603/telegram-cli/internal/account"
)

func newCapturingLogger(buf *bytes.Buffer) *zap.Logger {
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(buf), zapcore.DebugLevel)
	return zap.New(core, zap.WithCaller(false))
}

func TestWarnLoosePerms_warnsOnLooseMeta(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := account.AccountDir("alice")
	require.NoError(t, os.MkdirAll(dir, 0700))
	metaPath := filepath.Join(dir, "account.json")
	require.NoError(t, os.WriteFile(metaPath, []byte(`{}`), 0600))
	require.NoError(t, os.Chmod(metaPath, 0644))

	var buf bytes.Buffer
	WarnLoosePerms(newCapturingLogger(&buf), &account.Account{
		Meta: account.Meta{Name: "alice"},
		Dir:  dir,
		Sess: account.SessionFile("alice"),
	})

	lines := bytes.TrimSpace(buf.Bytes())
	require.NotEmpty(t, lines, "expected a warn line")
	var obj map[string]any
	require.NoError(t, json.Unmarshal(lines, &obj))
	require.Equal(t, "warn", obj["level"])
	require.Equal(t, "file permissions looser than 0600", obj["msg"])
	require.Equal(t, "meta", obj["kind"])
	require.Equal(t, "alice", obj["account"])
}

func TestWarnLoosePerms_silentWhenStrict(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := account.AccountDir("alice")
	require.NoError(t, os.MkdirAll(dir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "account.json"),
		[]byte(`{}`), 0600))

	var buf bytes.Buffer
	WarnLoosePerms(newCapturingLogger(&buf), &account.Account{
		Meta: account.Meta{Name: "alice"},
		Dir:  dir,
		Sess: account.SessionFile("alice"),
	})
	require.Empty(t, bytes.TrimSpace(buf.Bytes()))
}

func TestWarnLoosePerms_missingSessionIsNotAWarning(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := account.AccountDir("alice")
	require.NoError(t, os.MkdirAll(dir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "account.json"),
		[]byte(`{}`), 0600))

	var buf bytes.Buffer
	WarnLoosePerms(newCapturingLogger(&buf), &account.Account{
		Meta: account.Meta{Name: "alice"},
		Dir:  dir,
		Sess: account.SessionFile("alice"),
	})
	require.Empty(t, bytes.TrimSpace(buf.Bytes()))
}
