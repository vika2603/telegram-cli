// Package session contains remote session command actions.
package session

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gotd/td/tg"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	tgsession "github.com/vika2603/telegram-cli/internal/telegram/session"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// FetchFunc loads Telegram authorizations.
type FetchFunc func(context.Context, *tg.Client) (*tg.AccountAuthorizations, error)

// ResetFunc terminates a single Telegram authorization hash.
type ResetFunc func(context.Context, *tg.Client, int64) error

// ResetAllFunc terminates a set of Telegram authorization hashes.
type ResetAllFunc func(context.Context, *tg.Client, []int64) error

// List loads and maps active remote sessions.
func List(ctx context.Context, api *tg.Client, fetch FetchFunc) ([]output.AccountSessionRow, error) {
	if fetch == nil {
		return nil, fmt.Errorf("%w: internal error: session list fetch function is not configured", command.ErrPrecondition)
	}
	auths, err := fetch(ctx, api)
	if err != nil {
		return nil, err
	}
	rows := make([]output.AccountSessionRow, 0, len(auths.Authorizations))
	for _, a := range auths.Authorizations {
		rows = append(rows, RowFromAuth(a))
	}
	return rows, nil
}

// TerminateRequest is the normalized request for `tg session revoke`.
type TerminateRequest struct {
	Hash      string
	AllOthers bool
	Yes       bool
	Prompter  ui.Prompter
}

// Terminate validates and terminates one or more remote sessions.
func Terminate(
	ctx context.Context,
	api *tg.Client,
	req TerminateRequest,
	fetch FetchFunc,
	reset ResetFunc,
	resetAll ResetAllFunc,
) (map[string]any, error) {
	if fetch == nil || reset == nil || resetAll == nil {
		return nil, fmt.Errorf("%w: internal error: session revoke functions are not configured", command.ErrPrecondition)
	}
	if err := ValidateTerminate(req); err != nil {
		return nil, err
	}

	auths, err := fetch(ctx, api)
	if err != nil {
		return nil, err
	}
	currentHash := currentAuthorizationHash(auths)
	if req.AllOthers {
		return terminateAllOthers(ctx, api, req, auths, currentHash, resetAll)
	}
	return terminateSingle(ctx, api, req, auths, currentHash, reset)
}

// ValidateTerminate checks options that do not require a Telegram client.
func ValidateTerminate(req TerminateRequest) error {
	if req.Hash == "" && !req.AllOthers {
		return fmt.Errorf("%w: provide a hash or --all-others", command.ErrUsage)
	}
	if req.Hash != "" && req.AllOthers {
		return fmt.Errorf("%w: cannot use <hash> and --all-others together", command.ErrUsage)
	}
	return nil
}

// RowFromAuth converts a Telegram authorization into the output row shape.
func RowFromAuth(a tg.Authorization) output.AccountSessionRow {
	return output.AccountSessionRow{
		Hash:            strconv.FormatInt(a.Hash, 10),
		Current:         a.Current,
		OfficialApp:     a.OfficialApp,
		PasswordPending: a.PasswordPending,
		DeviceModel:     a.DeviceModel,
		Platform:        a.Platform,
		SystemVersion:   a.SystemVersion,
		AppName:         a.AppName,
		AppVersion:      a.AppVersion,
		APIID:           a.APIID,
		Country:         a.Country,
		Region:          a.Region,
		IP:              a.IP,
		DateCreated:     time.Unix(int64(a.DateCreated), 0).UTC().Format(time.RFC3339),
		DateActive:      time.Unix(int64(a.DateActive), 0).UTC().Format(time.RFC3339),
		Unconfirmed:     a.Unconfirmed,
	}
}

func terminateSingle(
	ctx context.Context,
	api *tg.Client,
	req TerminateRequest,
	auths *tg.AccountAuthorizations,
	current int64,
	reset ResetFunc,
) (map[string]any, error) {
	hashInt, perr := strconv.ParseInt(req.Hash, 10, 64)
	if perr != nil {
		return nil, fmt.Errorf("%w: hash must be an integer: %s", command.ErrUsage, perr.Error())
	}
	target := findAuthorization(auths, hashInt)
	if target == nil {
		return nil, fmt.Errorf("%w: no session with hash %s; run `tg session list` to see available sessions", command.ErrUsage, req.Hash)
	}
	if target.Current || target.Hash == current {
		return nil, fmt.Errorf("%w: use `tg auth logout` to sign out this tg instance", tgsession.ErrCurrent)
	}
	if err := ui.ConfirmDestructive(req.Prompter, singlePrompt(target), req.Yes); err != nil {
		return nil, err
	}
	if err := reset(ctx, api, hashInt); err != nil {
		return nil, err
	}
	return map[string]any{
		"action": "terminate",
		"hash":   req.Hash,
		"device": target.DeviceModel,
		"count":  1,
	}, nil
}

func terminateAllOthers(
	ctx context.Context,
	api *tg.Client,
	req TerminateRequest,
	auths *tg.AccountAuthorizations,
	current int64,
	resetAll ResetAllFunc,
) (map[string]any, error) {
	var victims []tg.Authorization
	var kept tg.Authorization
	for _, a := range auths.Authorizations {
		if a.Current {
			kept = a
			continue
		}
		victims = append(victims, a)
	}
	if len(victims) == 0 {
		return nil, command.NewNoResultsError("no other sessions to terminate")
	}
	if err := ui.ConfirmDestructive(req.Prompter, allOthersPrompt(victims, kept), req.Yes); err != nil {
		return nil, err
	}
	hashes := make([]int64, len(victims))
	for i, v := range victims {
		hashes[i] = v.Hash
	}
	if err := resetAll(ctx, api, hashes); err != nil {
		return nil, err
	}
	return map[string]any{
		"action":     "terminate",
		"all_others": true,
		"count":      len(victims),
		"kept_hash":  strconv.FormatInt(current, 10),
	}, nil
}

func currentAuthorizationHash(auths *tg.AccountAuthorizations) int64 {
	for _, a := range auths.Authorizations {
		if a.Current {
			return a.Hash
		}
	}
	return 0
}

func findAuthorization(auths *tg.AccountAuthorizations, hash int64) *tg.Authorization {
	for i := range auths.Authorizations {
		if auths.Authorizations[i].Hash == hash {
			return &auths.Authorizations[i]
		}
	}
	return nil
}

func singlePrompt(target *tg.Authorization) string {
	return fmt.Sprintf(
		"Terminate this session?\n  device:      %s · %s\n  platform:    %s %s\n  location:    %s (%s)\n  created:     %s\n  last active: %s",
		target.DeviceModel, target.AppName,
		target.Platform, target.SystemVersion,
		coalesce(target.Country), coalesce(target.IP),
		time.Unix(int64(target.DateCreated), 0).UTC().Format(time.RFC3339),
		time.Unix(int64(target.DateActive), 0).UTC().Format(time.RFC3339),
	)
}

func allOthersPrompt(victims []tg.Authorization, kept tg.Authorization) string {
	var pb strings.Builder
	_, _ = fmt.Fprintf(&pb, "Terminate %d session(s)?\n", len(victims))
	for i, v := range victims {
		_, _ = fmt.Fprintf(&pb, "  [%d] %s · %s · %s (%s)\n", i+1, v.DeviceModel, v.AppName,
			coalesce(v.Country), coalesce(v.IP))
	}
	_, _ = fmt.Fprintf(&pb, "\n(keeping current session: %s · %s · %s)",
		kept.DeviceModel, kept.AppName, coalesce(kept.Country))
	return pb.String()
}

func coalesce(s string) string {
	if s == "" {
		return "?"
	}
	return s
}
