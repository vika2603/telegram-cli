package session

import (
	"context"

	"github.com/gotd/td/tg"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// FakeClient implements Client for tests. Every method returns
// command.ErrUnsupported unless the matching *Fn override is set.
type FakeClient struct {
	SelfValue     tg.User
	InvokerFn     func() tg.Invoker
	ResolvePeerFn func(context.Context, ref.Ref) (tg.InputPeerClass, error)
	RefreshPeerFn func(context.Context, ref.Ref) (tg.InputPeerClass, error)
}

func (f *FakeClient) Invoker() tg.Invoker {
	if f.InvokerFn != nil {
		return f.InvokerFn()
	}
	return nil
}

func (f *FakeClient) Self() tg.User { return f.SelfValue }

func (f *FakeClient) ResolvePeer(ctx context.Context, r ref.Ref) (tg.InputPeerClass, error) {
	if f.ResolvePeerFn != nil {
		return f.ResolvePeerFn(ctx, r)
	}
	return nil, command.ErrUnsupported
}

func (f *FakeClient) RefreshPeer(ctx context.Context, r ref.Ref) (tg.InputPeerClass, error) {
	if f.RefreshPeerFn != nil {
		return f.RefreshPeerFn(ctx, r)
	}
	return nil, command.ErrUnsupported
}

var _ Client = (*FakeClient)(nil)
