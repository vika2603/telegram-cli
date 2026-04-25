package unpin_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/msg/pin"
	"github.com/vika2603/telegram-cli/internal/cli/msg/unpin"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func TestNew_UnpinDefaults(t *testing.T) {
	var captured *pin.Options
	f := runtime.NewTestInvocation(t)
	cmd := unpin.New(f, func(o *pin.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@a:5"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "@a:5", captured.RawMessageRef)
	require.True(t, captured.Unpin)
}
