package account_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/account/accounttest"
)

func TestRenameAccount_MovesSlotAndUpdatesName(t *testing.T) {
	root := accounttest.TempConfigRoot(t)
	accounttest.SeedAccount(t, root, "old", account.StateAUTHED)

	require.NoError(t, account.RenameAccount("old", "new"))

	_, err := account.ReadMeta("old")
	require.ErrorIs(t, err, account.ErrAccountNotFound)

	m, err := account.ReadMeta("new")
	require.NoError(t, err)
	require.Equal(t, "new", m.Name)
}

func TestRenameAccount_RejectsExistingTarget(t *testing.T) {
	root := accounttest.TempConfigRoot(t)
	accounttest.SeedAccount(t, root, "a", account.StateAUTHED)
	accounttest.SeedAccount(t, root, "b", account.StateAUTHED)

	require.ErrorIs(t, account.RenameAccount("a", "b"), account.ErrAccountExists)
}

func TestRenameAccount_RejectsMissingSource(t *testing.T) {
	accounttest.TempConfigRoot(t)
	require.ErrorIs(t, account.RenameAccount("ghost", "new"), account.ErrAccountNotFound)
}
