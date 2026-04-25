// Package auth contains auth-oriented command actions.
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/authflow"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/config"
)

// LoginRequest is the normalized request for `tg auth login`.
type LoginRequest struct {
	Name           string
	Force          bool
	QR             bool
	NoLogin        bool
	APIID          int
	APIHash        string
	APIIDChanged   bool
	APIHashChanged bool
}

// LoginFunc runs the Telegram login flow after the local slot is ready.
type LoginFunc func(context.Context, authflow.LoginOptions) error

// LoginDeps are the local account and credential dependencies used by Login.
type LoginDeps struct {
	Config               func() (*config.Config, error)
	ReadMeta             func(string) (account.Meta, error)
	AddAccount           func(account.Meta) error
	PromptAPICredentials func() (int, string, error)
	DoLogin              LoginFunc
}

// Login validates the request, creates the local slot when needed, runs the
// login flow when requested, and returns the final account DTO.
func Login(ctx context.Context, req LoginRequest, deps LoginDeps) (account.AccountDTO, error) {
	if !account.IsValidName(req.Name) {
		return account.AccountDTO{}, fmt.Errorf("%w: invalid account name %q", command.ErrUsage, req.Name)
	}
	if deps.Config == nil {
		return account.AccountDTO{}, fmt.Errorf("%w: auth login called without config function", command.ErrPrecondition)
	}
	if deps.ReadMeta == nil {
		return account.AccountDTO{}, fmt.Errorf("%w: auth login called without read-meta function", command.ErrPrecondition)
	}
	if deps.AddAccount == nil {
		return account.AccountDTO{}, fmt.Errorf("%w: auth login called without add-account function", command.ErrPrecondition)
	}

	cfg, err := deps.Config()
	if err != nil {
		return account.AccountDTO{}, err
	}
	def := defaultAccount(cfg)

	_, metaErr := deps.ReadMeta(req.Name)
	slotMissing := errors.Is(metaErr, account.ErrAccountNotFound)
	if metaErr != nil && !slotMissing {
		return account.AccountDTO{}, metaErr
	}

	if slotMissing {
		apiID, apiHash, err := createSlotCreds(req, cfg, deps.PromptAPICredentials)
		if err != nil {
			return account.AccountDTO{}, err
		}
		if err := deps.AddAccount(account.Meta{
			Name:    req.Name,
			State:   account.StateNEW,
			APIID:   apiID,
			APIHash: apiHash,
		}); err != nil {
			return account.AccountDTO{}, err
		}
	}

	if req.NoLogin {
		return readAccountDTO(req.Name, def, deps.ReadMeta)
	}

	meta, err := deps.ReadMeta(req.Name)
	if err != nil {
		return account.AccountDTO{}, err
	}
	if meta.State == account.StateAUTHED && !req.Force {
		return account.DTOFromMeta(meta, req.Name == def), nil
	}
	if deps.DoLogin == nil {
		return account.AccountDTO{}, fmt.Errorf("%w: auth login called without login function", command.ErrPrecondition)
	}
	if err := deps.DoLogin(ctx, authflow.LoginOptions{
		Name:           req.Name,
		QR:             req.QR,
		Force:          req.Force,
		APIID:          req.APIID,
		APIHash:        req.APIHash,
		APIIDChanged:   req.APIIDChanged,
		APIHashChanged: req.APIHashChanged,
	}); err != nil {
		return account.AccountDTO{}, err
	}
	return readAccountDTO(req.Name, def, deps.ReadMeta)
}

func createSlotCreds(req LoginRequest, cfg *config.Config, prompt func() (int, string, error)) (int, string, error) {
	if req.NoLogin {
		apiID, apiHash := authflow.CredsIfAvailableFromPtrs(cfg.APIID, cfg.APIHash)
		return apiID, apiHash, nil
	}
	apiID, apiHash, err := authflow.ResolveCreds(cfg.APIID, cfg.APIHash)
	if errors.Is(err, command.ErrPrecondition) {
		if prompt == nil {
			return 0, "", fmt.Errorf("%w: auth login called without API credential prompt", command.ErrPrecondition)
		}
		apiID, apiHash, err = prompt()
	}
	return apiID, apiHash, err
}

func readAccountDTO(name, def string, readMeta func(string) (account.Meta, error)) (account.AccountDTO, error) {
	meta, err := readMeta(name)
	if err != nil {
		return account.AccountDTO{}, err
	}
	return account.DTOFromMeta(meta, name == def), nil
}

func defaultAccount(cfg *config.Config) string {
	if cfg != nil && cfg.DefaultAccount != nil {
		return *cfg.DefaultAccount
	}
	return ""
}
