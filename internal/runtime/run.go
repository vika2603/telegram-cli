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
	return session.Options{
		APIID:   acct.Meta.APIID,
		APIHash: acct.Meta.APIHash,
		Device: session.DeviceOptions{
			Model:          "tg CLI",
			SystemVersion:  fmt.Sprintf("%s/%s", goruntime.GOOS, goruntime.GOARCH),
			AppVersion:     appVersion,
			SystemLangCode: "en",
			LangCode:       "en",
		},
	}
}
