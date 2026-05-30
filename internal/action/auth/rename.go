package auth

import (
	"errors"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/command"
)

// RenameRequest is the normalized request for `tg auth rename`.
type RenameRequest struct {
	Old string
	New string
}

// RenameDeps are the local account dependencies used by Rename.
type RenameDeps struct {
	ReadMeta func(string) (account.Meta, error)
	Rename   func(old, newName string) error
	// DaemonInstalled reports whether a host daemon is registered for the
	// account. Renaming under an installed daemon would orphan its
	// service, so Rename refuses in that case.
	DaemonInstalled func(string) bool
	// CurrentDefault is config.default_account; when it points at Old,
	// Rename repoints it to New via SetDefault.
	CurrentDefault string
	SetDefault     func(string) error
}

// RenameResult is the normalized result for `tg auth rename`.
type RenameResult struct {
	DTO      account.AccountDTO
	WarnName string
}

// Rename validates the slots, refuses when a daemon is installed for Old,
// renames the slot, repoints default_account when needed, and returns the
// renamed account's DTO.
func Rename(req RenameRequest, deps RenameDeps) (RenameResult, error) {
	if !account.IsValidName(req.New) {
		return RenameResult{}, fmt.Errorf("%w: invalid account name %q", command.ErrUsage, req.New)
	}
	if req.Old == req.New {
		return RenameResult{}, fmt.Errorf("%w: new name is the same as the old name", command.ErrUsage)
	}
	if deps.ReadMeta == nil || deps.Rename == nil {
		return RenameResult{}, fmt.Errorf("%w: auth rename called without required functions", command.ErrPrecondition)
	}

	if _, err := deps.ReadMeta(req.Old); err != nil {
		if errors.Is(err, account.ErrAccountNotFound) {
			return RenameResult{}, fmt.Errorf("%w: account %q does not exist", command.ErrUsage, req.Old)
		}
		return RenameResult{}, err
	}
	if _, err := deps.ReadMeta(req.New); err == nil {
		return RenameResult{}, fmt.Errorf("%w: account %q already exists", command.ErrUsage, req.New)
	}
	if deps.DaemonInstalled != nil && deps.DaemonInstalled(req.Old) {
		return RenameResult{}, fmt.Errorf("%w: account %q has a daemon installed; run `tg daemon uninstall --account %s` first", command.ErrPrecondition, req.Old, req.Old)
	}

	if err := deps.Rename(req.Old, req.New); err != nil {
		return RenameResult{}, err
	}
	if deps.CurrentDefault == req.Old && deps.SetDefault != nil {
		if err := deps.SetDefault(req.New); err != nil {
			return RenameResult{}, err
		}
	}

	meta, err := deps.ReadMeta(req.New)
	if err != nil {
		return RenameResult{}, err
	}
	return RenameResult{
		DTO:      account.DTOFromMeta(meta, deps.CurrentDefault == req.Old),
		WarnName: req.New,
	}, nil
}
