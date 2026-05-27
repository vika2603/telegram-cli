package logs_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/daemon/logs"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestRun_RejectsEmptyAccount(t *testing.T) {
	ios, _, _, _ := ui.Test()
	err := logs.Run(context.Background(), &logs.Options{IOStreams: ios})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_ReturnsPreconditionWhenMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ios, _, _, _ := ui.Test()
	err := logs.Run(context.Background(), &logs.Options{
		Account:   "alice",
		IOStreams: ios,
	})
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_PrintsTailFromExplicitLogFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "tail.log")
	require.NoError(t, os.WriteFile(logPath, []byte("a\nb\nc\nd\ne\n"), 0o600))

	ios, _, stdout, _ := ui.Test()
	err := logs.Run(context.Background(), &logs.Options{
		Account:   "alice",
		LogFile:   logPath,
		Lines:     3,
		IOStreams: ios,
	})
	require.NoError(t, err)

	got := strings.TrimSpace(stdout.String())
	require.Equal(t, "c\nd\ne", got)
}

func TestRun_HandlesShortFiles(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "tail.log")
	require.NoError(t, os.WriteFile(logPath, []byte("only line\n"), 0o600))

	ios, _, stdout, _ := ui.Test()
	require.NoError(t, logs.Run(context.Background(), &logs.Options{
		Account:   "alice",
		LogFile:   logPath,
		Lines:     100,
		IOStreams: ios,
	}))
	require.Equal(t, "only line\n", stdout.String())
}
