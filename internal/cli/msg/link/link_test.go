package link_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/cli/msg/link"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_Args(t *testing.T) {
	var captured *link.Options
	f := runtime.NewTestInvocation(t)
	cmd := link.New(f, func(o *link.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs([]string{"@news:42"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "@news:42", captured.RawMessageRef)
}

func TestRun_EmitsPublicChannelLink(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &link.Options{
		RawMessageRef: "@news:42",
		IOStreams:     ios,
		Resolve: func(context.Context, actionmessage.LinkQuery) (actionmessage.LinkPeer, error) {
			return actionmessage.LinkPeer{Username: "news", ChannelID: 100, IsChannel: true}, nil
		},
	}
	require.NoError(t, link.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "t.me/news/42")
}

func TestRun_EmitsPrivateChannelLink(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &link.Options{
		RawMessageRef: "c:123:456:7",
		IOStreams:     ios,
		Resolve: func(context.Context, actionmessage.LinkQuery) (actionmessage.LinkPeer, error) {
			return actionmessage.LinkPeer{Username: "", ChannelID: 123, IsChannel: true}, nil
		},
	}
	require.NoError(t, link.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "t.me/c/123/7")
}

func TestRun_UserChatIsNoLink(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &link.Options{
		RawMessageRef: "@alice:1",
		IOStreams:     ios,
		Resolve: func(context.Context, actionmessage.LinkQuery) (actionmessage.LinkPeer, error) {
			return actionmessage.LinkPeer{Username: "alice", IsChannel: false}, nil
		},
	}
	err := link.Run(context.Background(), opts)
	require.Error(t, err)
	require.ErrorIs(t, err, actionmessage.ErrNoLink)
}
