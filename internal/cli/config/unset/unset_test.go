package unset_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/config/unset"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// newOpts builds a base Options with a fresh temp dir config path and a
// test IOStreams. The stdout buffer is also returned for inspection.
func newOpts(t *testing.T) (*unset.Options, *bytes.Buffer) {
	t.Helper()
	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	return &unset.Options{F: f}, stdout
}

func TestRun_UnknownKey_IsUsage(t *testing.T) {
	opts, _ := newOpts(t)
	opts.Key = "nope"
	err := unset.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_KeyNotSet_NoResults(t *testing.T) {
	opts, _ := newOpts(t)
	// Write config that does NOT include the target key.
	require.NoError(t, os.WriteFile(opts.F.ConfigPath, []byte("version = 1\n"), 0600))

	opts.Key = "output.format"
	err := unset.Run(context.Background(), opts)
	require.Error(t, err)
	var nre *command.NoResultsError
	require.ErrorAs(t, err, &nre)
	require.Contains(t, err.Error(), "output.format")

	// File on disk must be unchanged.
	raw, readErr := config.ReadRawAt(opts.F.ConfigPath)
	require.NoError(t, readErr)
	_, present := config.ReadRaw(raw, "output.format")
	require.False(t, present)
}

func TestRun_NormalUnset_RemovesKey(t *testing.T) {
	opts, _ := newOpts(t)
	require.NoError(t, os.WriteFile(opts.F.ConfigPath, []byte("version = 1\n[output]\nformat = \"ndjson\"\n"), 0600))

	opts.Key = "output.format"
	require.NoError(t, unset.Run(context.Background(), opts))

	raw, err := config.ReadRawAt(opts.F.ConfigPath)
	require.NoError(t, err)
	_, present := config.ReadRaw(raw, "output.format")
	require.False(t, present, "output.format must be absent after unset")

	// version must still be present.
	v, ok := raw["version"]
	require.True(t, ok, "version must be preserved")
	require.EqualValues(t, 1, v)
}

func TestRun_PreservesUnknownKeys(t *testing.T) {
	opts, _ := newOpts(t)
	body := "version = 1\n[output]\nformat = \"json\"\n\n[foo]\nbar = 1\n"
	require.NoError(t, os.WriteFile(opts.F.ConfigPath, []byte(body), 0600))

	opts.Key = "output.format"
	require.NoError(t, unset.Run(context.Background(), opts))

	raw, err := config.ReadRawAt(opts.F.ConfigPath)
	require.NoError(t, err)
	v, ok := config.ReadRaw(raw, "foo.bar")
	require.True(t, ok, "foo.bar must be preserved after unset")
	require.EqualValues(t, 1, v)
}

func TestRun_APIHash_DeclinedPrompt_IsNotConfirmed(t *testing.T) {
	opts, _ := newOpts(t)
	opts.F.Prompter = &ui.StubPrompter{Answers: []any{false}}
	opts.Key = "api_hash"
	opts.Yes = false
	err := unset.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrNotConfirmed)
}

func TestRun_APIHash_AcceptedPrompt_Removes(t *testing.T) {
	opts, stdout := newOpts(t)
	opts.F.Prompter = &ui.StubPrompter{Answers: []any{true}}

	const realHash = "abcdefghijklmnopqrstuvwxyz123456" // 32 chars
	require.NoError(t, os.WriteFile(opts.F.ConfigPath, []byte("version = 1\napi_hash = \""+realHash+"\"\n"), 0600))

	opts.Key = "api_hash"
	opts.Yes = true
	require.NoError(t, unset.Run(context.Background(), opts))

	// api_hash must be absent on disk.
	raw, err := config.ReadRawAt(opts.F.ConfigPath)
	require.NoError(t, err)
	_, present := config.ReadRaw(raw, "api_hash")
	require.False(t, present, "api_hash must be absent after unset")

	// Row output must have redacted Old.
	var row output.ConfigKeyRow
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &row))
	require.Equal(t, "config_unset", row.Action)
	require.Equal(t, "api_hash", row.Key)
	oldStr, ok := row.Old.(string)
	require.True(t, ok)
	require.Equal(t, realHash[:8]+"…****", oldStr)
}

func TestRun_RowOldValueIncluded(t *testing.T) {
	opts, stdout := newOpts(t)
	require.NoError(t, os.WriteFile(opts.F.ConfigPath, []byte("version = 1\napi_id = 42\n"), 0600))

	opts.Key = "api_id"
	require.NoError(t, unset.Run(context.Background(), opts))

	var row output.ConfigKeyRow
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &row))
	require.Equal(t, "config_unset", row.Action)
	require.Equal(t, "api_id", row.Key)
	// TOML decodes integers as int64; JSON round-trips them as float64 via
	// json.Unmarshal into any — use EqualValues to compare across numeric types.
	require.EqualValues(t, int64(42), row.Old)
}

// Compile-time checks.
var _ = unset.Run
var _ = unset.Options{}

// filepath is used above; keep import via explicit reference.
var _ = filepath.Join
