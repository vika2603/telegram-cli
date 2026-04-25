package runtime

import (
	"context"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Invocation carries every dependency a command needs. IOStreams and Prompter
// are eager (set at construction). Config, Logger, Account, WithClient are
// lazy closures — invoked only when the command actually needs them.
//
// The Client lifetime uses a callback (WithClient) rather than a handle
// return because gotd resources are scoped to a single Run call and must be
// torn down regardless of error path. The callback form makes this structural.
type Invocation struct {
	AppVersion     string
	ExecutablePath string

	IOStreams *ui.IOStreams
	Prompter  ui.Prompter

	// ConfigPath and AccountName mirror the --config and --account
	// persistent root flags. They are populated by the root command's
	// PersistentPreRun; Invocation closures read them instead of reaching
	// back into cobra, so invocation wiring stays cobra-free.
	ConfigPath  string
	AccountName string

	Config  func() (*config.Config, error)
	Logger  func() (*zap.Logger, func(), error)
	Account func(name string) (*account.Account, error)

	WithClient func(
		ctx context.Context,
		acct *account.Account,
		opts session.Options,
		fn func(context.Context, session.Client) error,
	) error

	Resolver func(
		ctx context.Context,
		acct *account.Account,
		api *tg.Client,
	) (*peer.Resolver, error)

	WithPeers func(
		ctx context.Context,
		acct *account.Account,
		opts session.Options,
		fn func(ctx context.Context, api *tg.Client, pm *peers.Manager, res *peer.Resolver) error,
	) error
}

// NewInvocation builds the production Invocation with IOStreams wired to
// System() and Prompter wired to SystemPrompter. Config/Logger/Account/
// WithClient closures are populated by the root command's wiring (see
// internal/cli/root), not here — Invocation itself does not know which
// config path, account override, or logger settings to use until the
// root has parsed its flags.
func NewInvocation(appVersion string) *Invocation {
	ios := ui.System()
	return &Invocation{
		AppVersion: appVersion,
		IOStreams:  ios,
		Prompter:   &ui.SystemPrompter{IO: ios},
	}
}
