package runtime

import (
	"context"
	"fmt"
	goruntime "runtime"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
)

// DefaultWithClient is the production wiring for Invocation.WithClient. It
// delegates to session.Run, which owns the gotd lifecycle.
func DefaultWithClient(
	ctx context.Context,
	acct *account.Account,
	opts session.Options,
	fn func(context.Context, session.Client) error,
) error {
	return session.Run(ctx, acct, opts, fn)
}

// ClientOptsFrom maps account metadata to session.Options. Callers that need
// additional knobs should extend this function rather than constructing
// Options inline.
func ClientOptsFrom(f *Invocation, acct *account.Account) session.Options {
	appVersion := "dev"
	if f != nil && f.AppVersion != "" {
		appVersion = f.AppVersion
	}
	mode, maxSec := floodPolicyFrom(f)
	return session.Options{
		APIID:       acct.Meta.APIID,
		APIHash:     acct.Meta.APIHash,
		FloodMode:   mode,
		FloodMaxSec: maxSec,
		Device: session.DeviceOptions{
			Model:          "tg CLI",
			SystemVersion:  fmt.Sprintf("%s/%s", goruntime.GOOS, goruntime.GOARCH),
			AppVersion:     appVersion,
			SystemLangCode: "en",
			LangCode:       "en",
		},
	}
}

// floodPolicyFrom resolves the flood-wait mode + cap with the standard
// precedence: --flood-wait-max flag (cap only) > env/file config >
// hard defaults (fail mode, 30s cap). The env > file > default merge is
// already done inside f.Config(); this layer applies the flag on top.
//
// Config errors are swallowed — a malformed config must not stop a
// command from running; it just falls back to the safe defaults.
func floodPolicyFrom(f *Invocation) (session.FloodMode, int) {
	mode := session.FloodFail
	maxSec := 30
	if f != nil && f.Config != nil {
		if cfg, err := f.Config(); err == nil && cfg != nil {
			if cfg.FloodWait.Mode != nil && *cfg.FloodWait.Mode == "wait" {
				mode = session.FloodWait
			}
			if cfg.FloodWait.MaxSeconds != nil {
				maxSec = *cfg.FloodWait.MaxSeconds
			}
		}
	}
	if f != nil && f.FloodWaitMax != nil {
		maxSec = *f.FloodWaitMax
	}
	// --wait / --no-wait flag overrides config mode (flag > env > file).
	if f != nil && f.WaitFlood != nil {
		if *f.WaitFlood {
			mode = session.FloodWait
		} else {
			mode = session.FloodFail
		}
	}
	return mode, maxSec
}
