package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	actionauth "github.com/vika2603/telegram-cli/internal/action/auth"
	"github.com/vika2603/telegram-cli/internal/command"
)

// renameReadMeta builds a ReadMeta that knows about the given existing slots.
func renameReadMeta(existing ...string) func(string) (account.Meta, error) {
	set := map[string]bool{}
	for _, n := range existing {
		set[n] = true
	}
	return func(name string) (account.Meta, error) {
		if set[name] {
			return account.Meta{Name: name, State: account.StateAUTHED}, nil
		}
		return account.Meta{}, account.ErrAccountNotFound
	}
}

func TestRename_SuccessRepointsDefault(t *testing.T) {
	renamed := false
	defaultSet := ""
	res, err := actionauth.Rename(actionauth.RenameRequest{Old: "a", New: "b"}, actionauth.RenameDeps{
		ReadMeta: func(name string) (account.Meta, error) {
			if name == "a" || (name == "b" && renamed) {
				return account.Meta{Name: name, State: account.StateAUTHED}, nil
			}
			return account.Meta{}, account.ErrAccountNotFound
		},
		Rename:          func(string, string) error { renamed = true; return nil },
		DaemonInstalled: func(string) bool { return false },
		CurrentDefault:  "a",
		SetDefault:      func(n string) error { defaultSet = n; return nil },
	})
	require.NoError(t, err)
	require.True(t, renamed)
	require.Equal(t, "b", defaultSet, "default_account must follow the rename")
	require.Equal(t, "b", res.DTO.Name)
	require.True(t, res.DTO.Default)
}

func TestRename_NonDefaultLeavesDefaultAlone(t *testing.T) {
	renamed := false
	res, err := actionauth.Rename(actionauth.RenameRequest{Old: "a", New: "b"}, actionauth.RenameDeps{
		ReadMeta: func(name string) (account.Meta, error) {
			if name == "a" || (name == "b" && renamed) {
				return account.Meta{Name: name, State: account.StateAUTHED}, nil
			}
			return account.Meta{}, account.ErrAccountNotFound
		},
		Rename:          func(string, string) error { renamed = true; return nil },
		DaemonInstalled: func(string) bool { return false },
		CurrentDefault:  "other",
		SetDefault:      func(string) error { t.Fatal("must not repoint a default that isn't the renamed slot"); return nil },
	})
	require.NoError(t, err)
	require.False(t, res.DTO.Default)
}

func TestRename_RejectsDaemonInstalled(t *testing.T) {
	_, err := actionauth.Rename(actionauth.RenameRequest{Old: "a", New: "b"}, actionauth.RenameDeps{
		ReadMeta:        renameReadMeta("a"),
		Rename:          func(string, string) error { t.Fatal("must not rename when a daemon is installed"); return nil },
		DaemonInstalled: func(string) bool { return true },
	})
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRename_RejectsExistingTarget(t *testing.T) {
	_, err := actionauth.Rename(actionauth.RenameRequest{Old: "a", New: "b"}, actionauth.RenameDeps{
		ReadMeta:        renameReadMeta("a", "b"),
		Rename:          func(string, string) error { t.Fatal("must not rename onto an existing slot"); return nil },
		DaemonInstalled: func(string) bool { return false },
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRename_RejectsMissingSource(t *testing.T) {
	_, err := actionauth.Rename(actionauth.RenameRequest{Old: "ghost", New: "b"}, actionauth.RenameDeps{
		ReadMeta: renameReadMeta(),
		Rename:   func(string, string) error { t.Fatal("must not rename a missing slot"); return nil },
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRename_RejectsSameName(t *testing.T) {
	_, err := actionauth.Rename(actionauth.RenameRequest{Old: "a", New: "a"}, actionauth.RenameDeps{
		ReadMeta: renameReadMeta("a"),
		Rename:   func(string, string) error { t.Fatal("must not rename to the same name"); return nil },
	})
	require.ErrorIs(t, err, command.ErrUsage)
}
