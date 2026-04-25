package command_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/command"
)

func TestFlagError_IsUsage(t *testing.T) {
	err := command.FlagErrorf("bad flag %s", "--foo")
	require.ErrorIs(t, err, command.ErrUsage)
	require.Contains(t, err.Error(), "bad flag --foo")
}

func TestFlagErrorWrap_PreservesInner(t *testing.T) {
	inner := errors.New("inner cause")
	err := command.FlagErrorWrap(inner)
	require.ErrorIs(t, err, command.ErrUsage)
	require.ErrorIs(t, err, inner)
}

func TestMutuallyExclusive_SingleTrueIsOK(t *testing.T) {
	require.NoError(t, command.MutuallyExclusive("cannot combine", true, false, false))
}

func TestMutuallyExclusive_TwoTrueFails(t *testing.T) {
	err := command.MutuallyExclusive("cannot combine --a and --b", true, true)
	require.Error(t, err)
	require.ErrorIs(t, err, command.ErrUsage)
	require.Contains(t, err.Error(), "cannot combine --a and --b")
}

func TestMutuallyExclusive_AllFalseIsOK(t *testing.T) {
	require.NoError(t, command.MutuallyExclusive("", false, false))
}

func TestSilentError_IsSentinel(t *testing.T) {
	require.Error(t, command.ErrSilent)
	err := command.ErrSilent
	require.ErrorIs(t, err, command.ErrSilent)
}

func TestCancelError_IsSentinel(t *testing.T) {
	require.Error(t, command.ErrCancel)
}

func TestNoResultsError_CarriesMessage(t *testing.T) {
	err := command.NewNoResultsError("no accounts")
	require.EqualError(t, err, "no accounts")
	var nre *command.NoResultsError
	require.ErrorAs(t, err, &nre)
}
