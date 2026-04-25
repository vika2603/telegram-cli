package runtime

import (
	"context"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
)

// DefaultResolver builds a peer.Resolver from an open client + account.
// Intended as the default value of Invocation.Resolver when the caller
// already has a tg.Client (i.e. from inside a WithClient callback).
//
// ctx and acct are accepted to satisfy the Invocation.Resolver field signature
// but are not used by this pure-construction implementation; future impls
// may use them to hydrate the resolver from cache or server lookups.
//
// The PeerStore is intentionally nil: session.Run already owns the peersDB
// handle for the same account, so opening it again would block on the bbolt
// file lock and return ErrBusy. ID-based lookups degrade to ErrCacheMiss,
// which is the current resolver contract.
//
// selfID is zero here because this closure does not have access to
// cl.Self(). Callers resolving RefKindMe/Saved must use Invocation.WithPeers
// instead — that path populates selfID from the live session.
func DefaultResolver(
	ctx context.Context,
	acct *account.Account,
	api *tg.Client,
) (*peer.Resolver, error) {
	_ = ctx
	_ = acct
	mgr := peers.Options{}.Build(api)
	return peer.New(mgr, nil, 0)
}

// DefaultWithPeers wraps session.Run with peers.Manager + peer.Resolver
// construction so command code can stay short.
//
// The PeerStore is intentionally nil for the same reason as DefaultResolver:
// session.Run opens peers.db and owns its bbolt handle for the lifetime of
// the callback; a second open from this goroutine would block and return
// ErrBusy. Commands that need cached ID lookups must warm the cache via a
// username or t.me ref first.
func DefaultWithPeers(
	ctx context.Context,
	acct *account.Account,
	opts session.Options,
	fn func(ctx context.Context, api *tg.Client, pm *peers.Manager, res *peer.Resolver) error,
) error {
	return session.Run(ctx, acct, opts, func(ctx context.Context, cl session.Client) error {
		api := tg.NewClient(cl.Invoker())
		store := cl.PeerStore()
		mgr := peers.Options{Storage: store, Cache: store}.Build(api)
		selfID := cl.Self().ID
		res, err := peer.New(mgr, store, selfID)
		if err != nil {
			return err
		}
		return fn(ctx, api, mgr, res)
	})
}
