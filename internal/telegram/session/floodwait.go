package session

import (
	"context"
	"time"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
)

// ApplyFloodPolicy inspects err and applies the FloodMode strategy:
//   - If err is not a FLOOD_WAIT from gotd, return it unchanged.
//   - FloodFail: return FloodWaitError immediately.
//   - FloodWait with maxSec == 0: sleep the required seconds then return nil.
//   - FloodWait with maxSec > 0: if required <= maxSec sleep and return nil,
//     else return FloodWaitError unchanged.
//
// Respects ctx cancellation during sleep (returns ctx.Err()).
func ApplyFloodPolicy(ctx context.Context, mode FloodMode, maxSec int, err error) error {
	if err == nil {
		return nil
	}
	d, ok := tgerr.AsFloodWait(err)
	if !ok {
		return err
	}
	sec := int(d.Seconds())
	if sec == 0 {
		sec = 1
	}
	if mode == FloodFail {
		return &FloodWaitError{Seconds: sec}
	}
	if maxSec > 0 && sec > maxSec {
		return &FloodWaitError{Seconds: sec}
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// FloodWaitMiddleware returns a gotd telegram.Middleware that applies
// the FloodMode policy to every MTProto call. It is the wiring that
// makes ApplyFloodPolicy (and the flood_wait.mode / flood_wait.max_seconds
// config) actually take effect — installed via telegram.Options.Middlewares
// in dial.go.
//
// Behaviour:
//   - non-FLOOD_WAIT errors pass through unchanged (ApplyFloodPolicy
//     returns them verbatim).
//   - FloodFail (or wait that exceeds maxSec): the call returns a typed
//     *FloodWaitError immediately, so status/output classify it as
//     flood_wait with retry_after_seconds.
//   - FloodWait within maxSec: sleep the server-requested duration, then
//     retry the same call. Repeats if the retry also floods (each
//     individual wait is still capped by maxSec).
//
// ctx cancellation during a wait aborts with ctx.Err().
func FloodWaitMiddleware(mode FloodMode, maxSec int) telegram.Middleware {
	return telegram.MiddlewareFunc(func(next tg.Invoker) telegram.InvokeFunc {
		return func(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
			for {
				err := next.Invoke(ctx, input, output)
				if err == nil {
					return nil
				}
				// ApplyFloodPolicy returns nil only when it successfully
				// waited out a flood within the cap — retry in that case.
				if policyErr := ApplyFloodPolicy(ctx, mode, maxSec, err); policyErr != nil {
					return policyErr
				}
			}
		}
	})
}
