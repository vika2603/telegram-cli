package command_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/command"
)

func TestMeta_RoundTrip(t *testing.T) {
	cmd := &cobra.Command{Use: "foo"}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	m := command.MetaFrom(cmd)
	require.True(t, m.NeedsAccount)
	require.True(t, m.NeedsClient)
	require.False(t, m.SkipAuthCheck)
}

func TestMetaFrom_EmptyAnnotationsIsZero(t *testing.T) {
	cmd := &cobra.Command{Use: "foo"}
	m := command.MetaFrom(cmd)
	require.Equal(t, command.Meta{}, m)
}

func TestSetMeta_PreservesExistingAnnotations(t *testing.T) {
	cmd := &cobra.Command{Use: "foo", Annotations: map[string]string{"help:json": "true"}}
	command.SetMeta(cmd, command.Meta{NeedsClient: true})
	require.Equal(t, "true", cmd.Annotations["help:json"])
	require.Equal(t, "true", cmd.Annotations["meta:needs-client"])
}

func TestMeta_AccountFromArg_RoundTrip(t *testing.T) {
	cmd := &cobra.Command{Use: "logout"}
	command.SetMeta(cmd, command.Meta{AccountFromArg: true})
	m := command.MetaFrom(cmd)
	require.True(t, m.AccountFromArg)
	require.False(t, m.NeedsAccount)
	require.False(t, m.NeedsClient)
	require.False(t, m.SkipAuthCheck)
	require.False(t, m.LongRunning)
}
