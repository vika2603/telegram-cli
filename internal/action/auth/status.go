package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/output"
)

// StatusRequest is the normalized request for `tg auth status`.
type StatusRequest struct {
	Name  string
	Probe bool
}

// StatusProbeFunc runs a liveness check for an AUTHED account.
type StatusProbeFunc func(context.Context, *account.Account) error

// StatusDeps are the local account dependencies used by Status.
type StatusDeps struct {
	ResolveAccount  func(string) (*account.Account, error)
	Config          func() (*config.Config, error)
	SessionModified func(string) (string, error)
	Probe           StatusProbeFunc
}

// StatusResult is the normalized result for `tg auth status`.
type StatusResult struct {
	Row               output.AuthStatusRow
	ProbeSkipped      bool
	ProbeSkippedState string
}

// Status resolves an account and optionally probes an AUTHED session.
func Status(ctx context.Context, req StatusRequest, deps StatusDeps) (StatusResult, error) {
	if deps.ResolveAccount == nil {
		return StatusResult{}, fmt.Errorf("%w: auth status called without account resolver", command.ErrPrecondition)
	}
	if deps.SessionModified == nil {
		return StatusResult{}, fmt.Errorf("%w: auth status called without session stat function", command.ErrPrecondition)
	}

	acct, err := deps.ResolveAccount(req.Name)
	if err != nil {
		return StatusResult{}, err
	}
	name := acct.Meta.Name
	state := string(acct.Meta.State)
	sessionModified, err := deps.SessionModified(name)
	if err != nil {
		return StatusResult{}, err
	}

	result := StatusResult{
		Row: output.AuthStatusRow{
			Name:            name,
			State:           state,
			APIID:           acct.Meta.APIID,
			Default:         configuredDefault(deps.Config) == name,
			SessionModified: sessionModified,
		},
	}

	if req.Probe {
		result.Row.Probed = true
		if acct.Meta.State == account.StateAUTHED {
			if deps.Probe == nil {
				return StatusResult{}, fmt.Errorf("%w: auth status called without probe function", command.ErrPrecondition)
			}
			result.Row.ProbeOK = deps.Probe(ctx, acct) == nil
		} else {
			result.Row.ProbeOK = false
			result.ProbeSkipped = true
			result.ProbeSkippedState = state
		}
	}
	return result, nil
}

// DefaultSessionModified returns the RFC3339 mtime of session.bin or an empty
// string when the session file is absent.
func DefaultSessionModified(name string) (string, error) {
	info, err := os.Stat(account.SessionFile(name))
	if err == nil {
		return info.ModTime().UTC().Format(time.RFC3339), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return "", fmt.Errorf("stat session file for %s: %w", name, err)
}
