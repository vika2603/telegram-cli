package set_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/config/set"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// newOpts builds a base Options with a fresh temp dir config path and a
// test IOStreams. The stdout buffer is also returned for inspection.
func newOpts(t *testing.T) (*set.Options, *bytes.Buffer) {
	t.Helper()
	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	return &set.Options{F: f}, stdout
}

func TestRun_UnknownKey_IsUsage(t *testing.T) {
	opts, _ := newOpts(t)
	opts.Key = "nope"
	opts.Value = "anything"
	err := set.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_BadEnum_IsUsage(t *testing.T) {
	opts, _ := newOpts(t)
	opts.Key = "output.format"
	opts.Value = "garbage"
	err := set.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_BadInt_IsUsage(t *testing.T) {
	opts, _ := newOpts(t)
	opts.Key = "api_id"
	opts.Value = "not-a-number"
	err := set.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_APIHash_WithoutForce_IsUsage(t *testing.T) {
	opts, _ := newOpts(t)
	opts.Key = "api_hash"
	opts.Value = "abc123"
	opts.Force = false
	err := set.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
	require.Contains(t, err.Error(), "--force")
}

func TestRun_APIHash_ForceDeclinedPrompt_IsNotConfirmed(t *testing.T) {
	opts, _ := newOpts(t)
	opts.F.Prompter = &ui.StubPrompter{Answers: []any{false}}
	opts.Key = "api_hash"
	opts.Value = "abc123"
	opts.Force = true
	opts.Yes = false
	err := set.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrNotConfirmed)
}

func TestRun_NormalWrite_Success(t *testing.T) {
	opts, stdout := newOpts(t)
	// Pre-write a config with output.format = "json"
	require.NoError(t, os.WriteFile(opts.F.ConfigPath, []byte("version = 1\n[output]\nformat = \"json\"\n"), 0600))

	opts.Key = "output.format"
	opts.Value = "human"
	require.NoError(t, set.Run(context.Background(), opts))

	// Decode emitted JSON row
	var row output.ConfigKeyRow
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &row))
	require.Equal(t, "config_set", row.Action)
	require.Equal(t, "output.format", row.Key)
	require.Equal(t, "json", row.Old)
	require.Equal(t, "human", row.New)

	// Verify disk contents
	raw, err := config.ReadRawAt(opts.F.ConfigPath)
	require.NoError(t, err)
	v, ok := config.ReadRaw(raw, "output.format")
	require.True(t, ok)
	require.Equal(t, "human", v)

	// Verify file mode
	info, err := os.Stat(opts.F.ConfigPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestRun_PreservesUnknownKeys(t *testing.T) {
	opts, _ := newOpts(t)
	// Pre-write config with an unknown section alongside a standard key
	body := "version = 1\n[output]\nformat = \"json\"\n\n[foo]\nbar = 1\n"
	require.NoError(t, os.WriteFile(opts.F.ConfigPath, []byte(body), 0600))

	opts.Key = "output.format"
	opts.Value = "human"
	require.NoError(t, set.Run(context.Background(), opts))

	// Re-parse and assert foo.bar is still there
	raw, err := config.ReadRawAt(opts.F.ConfigPath)
	require.NoError(t, err)
	v, ok := config.ReadRaw(raw, "foo.bar")
	require.True(t, ok, "foo.bar must be preserved after set")
	require.EqualValues(t, 1, v)
}

func TestRun_APIHash_RedactedInRow(t *testing.T) {
	opts, stdout := newOpts(t)
	opts.F.Prompter = &ui.StubPrompter{Answers: []any{true}}

	const realHash = "abcdefghijklmnopqrstuvwxyz123456" // 32 chars
	opts.Key = "api_hash"
	opts.Value = realHash
	opts.Force = true
	opts.Yes = true
	require.NoError(t, set.Run(context.Background(), opts))

	// Row output must be redacted
	var row output.ConfigKeyRow
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &row))
	require.Equal(t, "config_set", row.Action)
	require.Equal(t, "api_hash", row.Key)
	newStr, ok := row.New.(string)
	require.True(t, ok)
	require.Equal(t, realHash[:8]+"…****", newStr)

	// Disk must contain the full real value
	raw, err := config.ReadRawAt(opts.F.ConfigPath)
	require.NoError(t, err)
	v, ok := config.ReadRaw(raw, "api_hash")
	require.True(t, ok)
	require.Equal(t, realHash, v)
}

func TestRun_VersionInjected_WhenMissing(t *testing.T) {
	opts, _ := newOpts(t)
	// Write config without version
	require.NoError(t, os.WriteFile(opts.F.ConfigPath, []byte("[output]\nformat = \"json\"\n"), 0600))

	opts.Key = "output.format"
	opts.Value = "human"
	require.NoError(t, set.Run(context.Background(), opts))

	raw, err := config.ReadRawAt(opts.F.ConfigPath)
	require.NoError(t, err)
	v, ok := raw["version"]
	require.True(t, ok, "version must be injected when absent")
	require.EqualValues(t, 1, v)
}

func TestRun_NormalWrite_RowFields(t *testing.T) {
	opts, stdout := newOpts(t)
	opts.Key = "log.level"
	opts.Value = "debug"
	require.NoError(t, set.Run(context.Background(), opts))

	var row output.ConfigKeyRow
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &row))
	require.Equal(t, "config_set", row.Action)
	require.Equal(t, "log.level", row.Key)
	require.Equal(t, "debug", row.New)
}

// Ensure StubPrompter is exported from ui (not a test-local type).
var _ ui.Prompter = (*ui.StubPrompter)(nil)
