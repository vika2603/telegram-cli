package me_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/me"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_CapturesOptions_JSONFlag(t *testing.T) {
	var captured *me.Options
	f := runtime.NewTestInvocation(t)
	cmd := me.New(f, func(o *me.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs([]string{"--json"})
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.NotNil(t, captured.Exporter)
}

func TestRun_RendersSelf(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &me.Options{
		Fetch: func(context.Context) (output.UserRow, error) {
			return output.UserRow{
				ID:        77,
				Username:  "me",
				FirstName: "Test",
				LastName:  "User",
				IsSelf:    true,
			}, nil
		},
		IOStreams: ios,
	}
	require.NoError(t, me.Run(context.Background(), opts))
	got := stdout.String()
	require.Contains(t, got, "Test User")
	require.Contains(t, got, "77")
}
