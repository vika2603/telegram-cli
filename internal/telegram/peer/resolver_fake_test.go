package peer_test

import (
	"context"

	"github.com/gotd/td/bin"
)

// stubInvoker satisfies tg.Invoker but never actually dispatches.
// Used only for tests that never hit the network (self, usage errors).
type stubInvoker struct{}

func (stubInvoker) Invoke(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
	return context.Canceled
}
