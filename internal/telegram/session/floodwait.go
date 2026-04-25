package session

import (
	"context"
	"time"

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
