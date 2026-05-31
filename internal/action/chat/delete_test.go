package chat_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestDeleteChat_ConfirmsBeforeDispatch(t *testing.T) {
	called := false
	_, err := actionchat.DeleteChat(context.Background(), actionchat.DeleteChatRequest{
		RawRef:   "@somegroup",
		Prompter: &ui.StubPrompter{Answers: []any{true}},
	}, func(_ context.Context, q actionchat.DeleteChatQuery) (output.PeerRef, error) {
		called = true
		require.Equal(t, "somegroup", q.Ref.Value)
		return output.PeerRef{}, nil
	})
	require.NoError(t, err)
	require.True(t, called)
}

func TestDeleteChat_DeclineSkipsDispatch(t *testing.T) {
	_, err := actionchat.DeleteChat(context.Background(), actionchat.DeleteChatRequest{
		RawRef:   "@somegroup",
		Prompter: &ui.StubPrompter{Answers: []any{false}},
	}, func(context.Context, actionchat.DeleteChatQuery) (output.PeerRef, error) {
		t.Fatal("delete must not run when the prompt is declined")
		return output.PeerRef{}, nil
	})
	require.ErrorIs(t, err, command.ErrNotConfirmed)
}
