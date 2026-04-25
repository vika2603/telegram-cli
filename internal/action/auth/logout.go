package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gotd/td/tg"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/output"
)

// LogoutRequest is the normalized request for `tg auth logout`.
type LogoutRequest struct {
	Name  string
	Purge bool
	Yes   bool
}

// LogoutDoArgs are passed into the server logout function.
type LogoutDoArgs struct {
	AcctName string
	API      *tg.Client
}

// LogoutDoFunc revokes the server session and performs AUTHED-slot cleanup.
type LogoutDoFunc func(context.Context, LogoutDoArgs) error

// RunAuthedLogoutFunc runs the server logout function with an authenticated client.
type RunAuthedLogoutFunc func(context.Context, *account.Account, LogoutDoFunc) error

// ClearDefaultResult describes a default-account clear attempt.
type ClearDefaultResult struct {
	Path    string
	Cleared bool
}

// LogoutDeps are the local account dependencies used by Logout.
type LogoutDeps struct {
	ResolveAccount func(string) (*account.Account, error)
	Config         func() (*config.Config, error)
	Confirm        func(message string, yes bool) error
	RunAuthed      RunAuthedLogoutFunc
	DeleteSession  func(string) error
	WriteMeta      func(account.Meta) error
	RemoveAccount  func(string) error
	ClearDefault   func() (ClearDefaultResult, error)
	Do             LogoutDoFunc
}

// LogoutResult is the normalized result for `tg auth logout`.
type LogoutResult struct {
	Row      output.LogoutRow
	Warnings []string
}

// Logout validates, confirms, revokes when needed, performs local cleanup, and
// returns the row that the CLI should render.
func Logout(ctx context.Context, req LogoutRequest, deps LogoutDeps) (LogoutResult, error) {
	if deps.Do == nil {
		return LogoutResult{}, fmt.Errorf("%w: auth logout called without Do", command.ErrPrecondition)
	}
	if deps.ResolveAccount == nil {
		return LogoutResult{}, fmt.Errorf("%w: auth logout called without account resolver", command.ErrPrecondition)
	}
	if deps.Confirm == nil {
		return LogoutResult{}, fmt.Errorf("%w: auth logout called without confirm function", command.ErrPrecondition)
	}
	if deps.DeleteSession == nil || deps.WriteMeta == nil {
		return LogoutResult{}, fmt.Errorf("%w: auth logout called without local cleanup functions", command.ErrPrecondition)
	}

	acct, err := deps.ResolveAccount(req.Name)
	if err != nil {
		return LogoutResult{}, err
	}
	name := acct.Meta.Name
	def := configuredDefault(deps.Config)

	if err := deps.Confirm(logoutPrompt(name, req.Purge, def == name), req.Yes); err != nil {
		return LogoutResult{}, err
	}

	if acct.Meta.State == account.StateAUTHED {
		if deps.RunAuthed == nil {
			return LogoutResult{}, fmt.Errorf("%w: auth logout called without authenticated runner", command.ErrPrecondition)
		}
		if err := deps.RunAuthed(ctx, acct, deps.Do); err != nil && !IsAlreadyLoggedOut(err) {
			return LogoutResult{}, err
		}
	} else if err := resetLocalSession(acct.Meta, deps); err != nil {
		return LogoutResult{}, err
	}

	defaultCleared := false
	var warnings []string
	if req.Purge {
		if deps.RemoveAccount == nil {
			return LogoutResult{}, fmt.Errorf("%w: auth logout called without remove-account function", command.ErrPrecondition)
		}
		if err := deps.RemoveAccount(name); err != nil {
			return LogoutResult{}, err
		}
		if def == name {
			if deps.ClearDefault == nil {
				return LogoutResult{}, fmt.Errorf("%w: auth logout called without clear-default function", command.ErrPrecondition)
			}
			clear, err := deps.ClearDefault()
			if err != nil {
				warnings = append(warnings, fmt.Sprintf(
					"tg: purge of %q succeeded but could not clear default_account in %s: %s; run `tg config unset default_account` manually.",
					name, clear.Path, err.Error()))
			} else {
				defaultCleared = clear.Cleared
			}
		}
	}

	return LogoutResult{
		Row: output.LogoutRow{
			Action:         "logout",
			Name:           name,
			Purged:         req.Purge,
			DefaultCleared: defaultCleared,
		},
		Warnings: warnings,
	}, nil
}

// DefaultLogoutDo is the production server logout and AUTHED-slot cleanup.
func DefaultLogoutDo(ctx context.Context, a LogoutDoArgs) error {
	if _, err := a.API.AuthLogOut(ctx); err != nil && !IsAlreadyLoggedOut(err) {
		return err
	}
	if err := account.DeleteSession(a.AcctName); err != nil {
		return err
	}
	meta, err := account.ReadMeta(a.AcctName)
	if err != nil {
		if errors.Is(err, account.ErrAccountNotFound) {
			return nil
		}
		return err
	}
	meta.State = account.StateNEW
	return account.WriteMeta(meta)
}

// IsAlreadyLoggedOut reports whether Telegram says the local session is gone.
func IsAlreadyLoggedOut(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "AUTH_KEY_UNREGISTERED") ||
		strings.Contains(msg, "AUTH_KEY_INVALID") ||
		strings.Contains(msg, "SESSION_REVOKED")
}

func logoutPrompt(name string, purge, clearsDefault bool) string {
	if !purge {
		return fmt.Sprintf("Log out account %q?", name)
	}
	msg := fmt.Sprintf(
		"Purge account %q? This revokes the server session AND removes local storage"+
			" (peers.db, session.bin, account.json, account.lock).",
		name,
	)
	if clearsDefault {
		msg += " (default_account pointer will be cleared automatically.)"
	}
	return msg
}

func configuredDefault(configFn func() (*config.Config, error)) string {
	if configFn == nil {
		return ""
	}
	cfg, err := configFn()
	if err != nil {
		return ""
	}
	return defaultAccount(cfg)
}

func resetLocalSession(meta account.Meta, deps LogoutDeps) error {
	_ = deps.DeleteSession(meta.Name)
	meta.State = account.StateNEW
	return deps.WriteMeta(meta)
}
