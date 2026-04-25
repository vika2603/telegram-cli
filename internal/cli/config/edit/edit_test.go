package edit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/config/edit"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// newOpts builds a base Options with a fresh temp-dir config path and a test
// IOStreams triple. The returned buffers are stdout and stderr for inspection.
func newOpts(t *testing.T) (*edit.Options, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	ios, _, stdout, stderr := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	return &edit.Options{F: f}, stdout, stderr
}

// shellWriter returns a ResolveEditor stub that overwrites the tmp file (passed
// as $1) with content via a shell here-doc.
func shellWriter(content string) func() ([]string, error) {
	return func() ([]string, error) {
		script := `cat > "$1" << 'EDITEOF'
` + content + `
EDITEOF`
		return []string{"/bin/sh", "-c", script, "editor"}, nil
	}
}

// TestRun_NoResolverReturnsPrecondition verifies that a nil ResolveEditor
// returns ErrPrecondition immediately.
func TestRun_NoResolverReturnsPrecondition(t *testing.T) {
	opts, _, _ := newOpts(t)
	opts.ResolveEditor = nil
	err := edit.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

// TestRun_EditorSucceedsAndValidTOML_WritesFile verifies that a valid TOML
// file produced by the editor is written to the live config path and that the
// output JSON reports the correct change count.
func TestRun_EditorSucceedsAndValidTOML_WritesFile(t *testing.T) {
	opts, stdout, _ := newOpts(t)
	opts.ResolveEditor = shellWriter("version = 1\napi_id = 99\n")

	require.NoError(t, edit.Run(context.Background(), opts))

	got, err := os.ReadFile(opts.F.ConfigPath)
	require.NoError(t, err)
	require.Contains(t, string(got), "api_id")
	require.Contains(t, string(got), "version")

	// Output must be a valid JSON object with action = "config_edit".
	var row map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &row))
	require.Equal(t, "config_edit", row["action"])
	// Initial file was empty; editor wrote version + api_id → 2 changed keys.
	require.EqualValues(t, 2, row["changes"])
}

// TestRun_EditorInvalidTOML_PromptDeclined verifies that invalid TOML triggers
// a re-open prompt and, when declined, returns ErrNotConfirmed without
// touching the live file.
func TestRun_EditorInvalidTOML_PromptDeclined(t *testing.T) {
	opts, _, _ := newOpts(t)
	opts.ResolveEditor = shellWriter("=====garbage====")
	opts.F.Prompter = &ui.StubPrompter{Answers: []any{false}}

	err := edit.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrNotConfirmed)

	// Live file must not have been created.
	_, statErr := os.Stat(opts.F.ConfigPath)
	require.ErrorIs(t, statErr, os.ErrNotExist, "live file must be absent after declined prompt")
}

// TestRun_EditorMissingVersion_PromptDeclined verifies that a TOML file that
// omits `version` causes config.Load to error, triggering the re-open prompt.
// When declined, ErrNotConfirmed is returned and the live file is unchanged.
func TestRun_EditorMissingVersion_PromptDeclined(t *testing.T) {
	opts, _, _ := newOpts(t)
	opts.ResolveEditor = shellWriter("api_id = 1\n")
	opts.F.Prompter = &ui.StubPrompter{Answers: []any{false}}

	err := edit.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrNotConfirmed)

	_, statErr := os.Stat(opts.F.ConfigPath)
	require.ErrorIs(t, statErr, os.ErrNotExist, "live file must be absent after declined prompt")
}

// TestRun_EditorBadEnum_PromptDeclined verifies that a TOML file with an
// invalid enum value (output.format = "xml") causes ValidateEnums to error,
// triggering the prompt. When declined, live file is unchanged.
func TestRun_EditorBadEnum_PromptDeclined(t *testing.T) {
	opts, _, _ := newOpts(t)
	opts.ResolveEditor = shellWriter("version = 1\n[output]\nformat = \"xml\"\n")
	opts.F.Prompter = &ui.StubPrompter{Answers: []any{false}}

	err := edit.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrNotConfirmed)

	_, statErr := os.Stat(opts.F.ConfigPath)
	require.ErrorIs(t, statErr, os.ErrNotExist, "live file must be absent after declined prompt")
}

// TestRun_EditorNonZeroExit verifies that a non-zero editor exit returns
// ErrUsage and leaves the live file untouched.
func TestRun_EditorNonZeroExit(t *testing.T) {
	opts, _, _ := newOpts(t)
	opts.ResolveEditor = func() ([]string, error) {
		return []string{"/bin/sh", "-c", "exit 1"}, nil
	}

	err := edit.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)

	_, statErr := os.Stat(opts.F.ConfigPath)
	require.ErrorIs(t, statErr, os.ErrNotExist, "live file must be absent after non-zero editor exit")
}

// TestRun_UnknownKeysWarnButAccept verifies that a valid TOML file containing
// an unknown section causes a warning on stderr but still succeeds: the live
// file is written and the output JSON is emitted.
func TestRun_UnknownKeysWarnButAccept(t *testing.T) {
	opts, stdout, stderr := newOpts(t)
	opts.ResolveEditor = shellWriter("version = 1\n\n[foo]\nbar = 1\n")

	require.NoError(t, edit.Run(context.Background(), opts))

	// Live file must exist and contain the [foo] section.
	got, err := os.ReadFile(opts.F.ConfigPath)
	require.NoError(t, err)
	require.Contains(t, string(got), "foo")

	// stderr must contain a "warning:" line for the unknown key.
	require.Contains(t, stderr.String(), "warning:")

	// stdout must have a valid JSON row.
	var row map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &row))
	require.Equal(t, "config_edit", row["action"])
}
