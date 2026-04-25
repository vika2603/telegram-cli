package setbio_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/profile/setbio"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	var captured *setbio.Options
	f := runtime.NewTestInvocation(t)
	cmd := setbio.New(f, func(o *setbio.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"hello world"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "hello world", captured.Bio)
}

func TestRun_NilUpdateClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &setbio.Options{Bio: "hi", IOStreams: ios, Stdin: ios.In}
	err := setbio.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_StubbedUpdatePositional(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &setbio.Options{
		Bio: "hi there", IOStreams: ios, Stdin: ios.In,
		Update: func(_ context.Context, s string) (output.ProfileRow, error) {
			require.Equal(t, "hi there", s)
			return output.ProfileRow{Action: "set-bio", Bio: s}, nil
		},
	}
	require.NoError(t, setbio.Run(context.Background(), opts))
	s := stdout.String()
	require.Contains(t, s, `"action":"set-bio"`)
	require.Contains(t, s, `"bio":"hi there"`)
}

func TestRun_StdinArg(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	ios.In = io.NopCloser(strings.NewReader("multi\nline\n"))
	opts := &setbio.Options{
		Bio: "-", IOStreams: ios, Stdin: ios.In,
		Update: func(_ context.Context, s string) (output.ProfileRow, error) {
			require.Equal(t, "multi\nline", s)
			return output.ProfileRow{Action: "set-bio", Bio: s}, nil
		},
	}
	require.NoError(t, setbio.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "multi")
}

func TestRun_StubbedUpdateEmpty(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &setbio.Options{
		Bio: "", IOStreams: ios, Stdin: ios.In,
		Update: func(_ context.Context, s string) (output.ProfileRow, error) {
			require.Empty(t, s)
			return output.ProfileRow{Action: "set-bio"}, nil
		},
	}
	require.NoError(t, setbio.Run(context.Background(), opts))
}
