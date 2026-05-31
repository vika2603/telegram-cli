package info_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/cli/chat/info"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_RequiresRef(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := info.New(f, nil)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)
}

func TestNew_ParsesRef(t *testing.T) {
	var captured *info.Options
	f := runtime.NewTestInvocation(t)
	cmd := info.New(f, func(o *info.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs([]string{"@durov"})
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.Equal(t, "@durov", captured.RawRef)
}

func TestRun_RendersRow(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &info.Options{
		RawRef:    "@durov",
		IOStreams: ios,
		Fetch: func(context.Context, actionchat.ShowQuery) (output.ChatRow, error) {
			return output.ChatRow{ID: 1, Kind: "user", Title: "Pavel", Username: "durov"}, nil
		},
	}
	require.NoError(t, info.Run(context.Background(), opts))
	got := stdout.String()
	require.Contains(t, got, "Pavel")
	require.Contains(t, got, "durov")
}

func TestRun_PeerNotFoundSurfaces(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &info.Options{
		RawRef:    "@missing",
		IOStreams: ios,
		Fetch: func(context.Context, actionchat.ShowQuery) (output.ChatRow, error) {
			return output.ChatRow{}, peer.ErrNotFound
		},
	}
	err := info.Run(context.Background(), opts)
	require.ErrorIs(t, err, peer.ErrNotFound)
}
