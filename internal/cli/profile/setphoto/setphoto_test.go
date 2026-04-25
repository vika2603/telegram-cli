package setphoto_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/profile/setphoto"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "a.jpg")
	require.NoError(t, os.WriteFile(tmp, []byte("x"), 0o600))

	var captured *setphoto.Options
	f := runtime.NewTestInvocation(t)
	cmd := setphoto.New(f, func(o *setphoto.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{tmp})
	require.NoError(t, cmd.Execute())
	require.Equal(t, tmp, captured.Path)
}

func TestRun_NilUploadClosureReturnsPrecondition(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "a.jpg")
	require.NoError(t, os.WriteFile(tmp, []byte("x"), 0o600))
	ios, _, _, _ := ui.Test()
	opts := &setphoto.Options{Path: tmp, IOStreams: ios}
	err := setphoto.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_MissingFile(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &setphoto.Options{
		Path: "/no/such", IOStreams: ios,
		Upload: func(_ context.Context, _ string, _ io.Reader) (output.ProfileRow, error) {
			t.Fatal("Upload must not run when file is missing")
			return output.ProfileRow{}, nil
		},
	}
	err := setphoto.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_StubbedUpload(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "a.jpg")
	require.NoError(t, os.WriteFile(tmp, []byte("x"), 0o600))
	ios, _, stdout, _ := ui.Test()
	opts := &setphoto.Options{
		Path: tmp, IOStreams: ios,
		Upload: func(_ context.Context, path string, _ io.Reader) (output.ProfileRow, error) {
			require.Equal(t, tmp, path)
			return output.ProfileRow{Action: "set-photo", PhotoID: 99}, nil
		},
	}
	require.NoError(t, setphoto.Run(context.Background(), opts))
	s := stdout.String()
	require.Contains(t, s, `"action":"set-photo"`)
	require.Contains(t, s, `"photo_id":99`)
}

func TestRun_StdinBypassesStat(t *testing.T) {
	ios, _, _, _ := ui.Test()
	called := false
	opts := &setphoto.Options{
		Path: "-", IOStreams: ios,
		Upload: func(_ context.Context, path string, stdin io.Reader) (output.ProfileRow, error) {
			called = true
			require.Equal(t, "-", path)
			require.NotNil(t, stdin)
			return output.ProfileRow{Action: "set-photo", PhotoID: 7}, nil
		},
	}
	require.NoError(t, setphoto.Run(context.Background(), opts))
	require.True(t, called)
}
