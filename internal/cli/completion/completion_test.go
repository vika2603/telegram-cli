package completion_test

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/completion"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// newStandaloneCmd returns a completion command with no parent.
// Used for tests that only care about arg validation errors.
func newStandaloneCmd(t *testing.T) *cobra.Command {
	t.Helper()
	f := runtime.NewTestInvocation(t)
	return completion.New(f)
}

// newRootedCmd attaches the completion command to a minimal root so that
// cmd.Root() resolves to a real root inside RunE. Returns root and the
// stdout buffer wired through IOStreams.
func newRootedCmd(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios

	cmd := completion.New(f)

	root := &cobra.Command{Use: "tg"}
	root.AddGroup(&cobra.Group{ID: "setup", Title: "Setup"})
	root.AddCommand(cmd)

	return root, stdout
}

func TestNew_ExactArgsRequired(t *testing.T) {
	cmd := newStandaloneCmd(t)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)
}

func TestNew_RejectsUnknownShell(t *testing.T) {
	cmd := newStandaloneCmd(t)
	cmd.SetArgs([]string{"tcsh"})
	err := cmd.Execute()
	require.Error(t, err)
}

func TestNew_BashWritesScript(t *testing.T) {
	root, stdout := newRootedCmd(t)
	root.SetArgs([]string{"completion", "bash"})
	require.NoError(t, root.Execute())
	assert.Contains(t, stdout.String(), "bash completion")
}

func TestNew_ZshWritesScript(t *testing.T) {
	root, stdout := newRootedCmd(t)
	root.SetArgs([]string{"completion", "zsh"})
	require.NoError(t, root.Execute())
	assert.Contains(t, stdout.String(), "#compdef")
}
