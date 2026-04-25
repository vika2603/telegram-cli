package auth

import (
	"errors"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/command"
)

// SwitchRequest is the normalized request for `tg auth switch`.
type SwitchRequest struct {
	Name string
}

// SwitchDeps are the local account dependencies used by Switch.
type SwitchDeps struct {
	ReadMeta   func(string) (account.Meta, error)
	SetDefault func(string) error
}

// SwitchResult is the normalized result for `tg auth switch`.
type SwitchResult struct {
	DTO      account.AccountDTO
	WarnName string
}

// Switch validates the slot, writes default_account, and returns its DTO.
func Switch(req SwitchRequest, deps SwitchDeps) (SwitchResult, error) {
	if !account.IsValidName(req.Name) {
		return SwitchResult{}, fmt.Errorf("%w: invalid account name %q", command.ErrUsage, req.Name)
	}
	if deps.ReadMeta == nil {
		return SwitchResult{}, fmt.Errorf("%w: auth switch called without read-meta function", command.ErrPrecondition)
	}
	if deps.SetDefault == nil {
		return SwitchResult{}, fmt.Errorf("%w: auth switch called without set-default function", command.ErrPrecondition)
	}

	meta, err := deps.ReadMeta(req.Name)
	if err != nil {
		if errors.Is(err, account.ErrAccountNotFound) {
			return SwitchResult{}, fmt.Errorf("%w: account %q does not exist", command.ErrUsage, req.Name)
		}
		return SwitchResult{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if err := deps.SetDefault(req.Name); err != nil {
		return SwitchResult{}, err
	}
	return SwitchResult{
		DTO:      account.DTOFromMeta(meta, true),
		WarnName: req.Name,
	}, nil
}
