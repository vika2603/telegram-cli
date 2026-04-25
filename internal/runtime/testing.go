package runtime

import (
	"context"
	"testing"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// NewTestInvocation returns an Invocation with harmless defaults:
//   - IOStreams.Test() buffer triple (all TTYs false)
//   - ui.StubPrompter with empty answer queue
//   - Config returning config.Defaults()
//   - Logger returning zap.NewNop()
//   - Account returning command.ErrPrecondition ("no account resolved")
//   - WithClient returning command.ErrUnsupported
//
// Tests override individual fields directly on the returned *Invocation.
func NewTestInvocation(t *testing.T) *Invocation {
	t.Helper()
	ios, _, _, _ := ui.Test()
	return &Invocation{
		AppVersion: "test",
		IOStreams:  ios,
		Prompter:   &ui.StubPrompter{},
		Config: func() (*config.Config, error) {
			d := config.Defaults()
			return &d, nil
		},
		Logger: func() (*zap.Logger, func(), error) {
			return zap.NewNop(), func() {}, nil
		},
		Account: func(string) (*account.Account, error) {
			return nil, command.ErrPrecondition
		},
		WithClient: func(_ context.Context, _ *account.Account, _ session.Options,
			_ func(context.Context, session.Client) error) error {
			return command.ErrUnsupported
		},
		Resolver: func(context.Context, *account.Account, *tg.Client) (*peer.Resolver, error) {
			return nil, command.ErrUnsupported
		},
		WithPeers: func(context.Context, *account.Account, session.Options,
			func(context.Context, *tg.Client, *peers.Manager, *peer.Resolver) error) error {
			return command.ErrUnsupported
		},
	}
}
