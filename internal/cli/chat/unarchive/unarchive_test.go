package unarchive_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/cli/chat/unarchive"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	var captured *unarchive.Options
	cmd := unarchive.New(f, func(o *unarchive.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@durov"})
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.Equal(t, "@durov", captured.RawRef)
}

func TestRun_NilDoReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &unarchive.Options{RawRef: "@durov", IOStreams: ios}
	err := unarchive.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
	require.ErrorContains(t, err, "chat unarchive called without do function")
}

func TestRun_InvalidRefIsUsage(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &unarchive.Options{
		RawRef:    "",
		IOStreams: ios,
		Do: func(_ context.Context, _ actionchat.FolderQuery) (output.ChatFolderRow, error) {
			t.Fatal("Do must not be called when ref is invalid")
			return output.ChatFolderRow{}, nil
		},
	}
	err := unarchive.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_StubbedOK(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &unarchive.Options{
		RawRef:    "@durov",
		IOStreams: ios,
		Do: func(_ context.Context, args actionchat.FolderQuery) (output.ChatFolderRow, error) {
			require.Equal(t, "durov", args.Ref.Value)
			return output.ChatFolderRow{Action: "unarchive", Folder: 0}, nil
		},
	}
	require.NoError(t, unarchive.Run(context.Background(), opts))
}
