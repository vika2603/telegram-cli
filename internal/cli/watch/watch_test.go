package watch_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/watch"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_CapturesRefsKindsAndLimit(t *testing.T) {
	f := &runtime.Invocation{IOStreams: ui.System()}

	var captured *watch.Options
	cmd := watch.New(f, func(o *watch.Options) error {
		captured = o
		return nil
	})

	cmd.SetArgs([]string{"@chan", "me", "--kind=message,edit", "--limit=3"})
	require.NoError(t, cmd.Execute())

	require.NotNil(t, captured)
	require.Equal(t, []string{"@chan", "me"}, captured.RawRefs)
	require.Equal(t, []string{"message", "edit"}, captured.Kinds)
	require.Equal(t, 3, captured.Limit)
}

func TestNew_DefaultLimitIsUnlimited(t *testing.T) {
	f := &runtime.Invocation{IOStreams: ui.System()}
	var captured *watch.Options
	cmd := watch.New(f, func(o *watch.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.Equal(t, 0, captured.Limit)
	require.Empty(t, captured.RawRefs)
}
