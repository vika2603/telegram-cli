package session

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// directClient is the concrete Client used for non-daemon mode.
type directClient struct {
	tgCl *telegram.Client
	self tg.User
	opts Options
	acct *account.Account
}

func (c *directClient) Invoker() tg.Invoker { return c.tgCl.API().Invoker() }
func (c *directClient) Self() tg.User       { return c.self }

func (c *directClient) ResolvePeer(_ context.Context, _ ref.Ref) (tg.InputPeerClass, error) {
	return nil, fmt.Errorf("%w: peer resolution not implemented yet", command.ErrUnsupported)
}

func (c *directClient) RefreshPeer(_ context.Context, _ ref.Ref) (tg.InputPeerClass, error) {
	return nil, fmt.Errorf("%w: peer resolution not implemented yet", command.ErrUnsupported)
}
