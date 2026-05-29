package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tgerr"
	"github.com/stretchr/testify/require"
)

func TestApplyFloodPolicy_failMode_returnsErrFloodWait(t *testing.T) {
	gerr := tgerr.New(420, "FLOOD_WAIT_5")
	got := ApplyFloodPolicy(context.Background(), FloodFail, 0, gerr)
	require.ErrorIs(t, got, ErrFloodWait)
}

func TestApplyFloodPolicy_waitMode_withinCap(t *testing.T) {
	gerr := tgerr.New(420, "FLOOD_WAIT_1")
	start := time.Now()
	got := ApplyFloodPolicy(context.Background(), FloodWait, 5, gerr)
	require.NoError(t, got)
	require.GreaterOrEqual(t, time.Since(start), 900*time.Millisecond)
}

func TestApplyFloodPolicy_waitMode_overCap_returnsErr(t *testing.T) {
	gerr := tgerr.New(420, "FLOOD_WAIT_60")
	got := ApplyFloodPolicy(context.Background(), FloodWait, 5, gerr)
	require.ErrorIs(t, got, ErrFloodWait)
}

func TestApplyFloodPolicy_notAFloodWait_passthrough(t *testing.T) {
	other := errors.New("something else")
	got := ApplyFloodPolicy(context.Background(), FloodWait, 5, other)
	require.Equal(t, other, got)
}

func TestApplyFloodPolicy_ctxCancelled_returnsCtxErr(t *testing.T) {
	gerr := tgerr.New(420, "FLOOD_WAIT_10")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := ApplyFloodPolicy(ctx, FloodWait, 60, gerr)
	require.Equal(t, context.Canceled, got)
}

// fakeInvoker counts calls and returns scripted errors so the
// middleware's retry loop can be exercised without a real client.
type fakeInvoker struct {
	calls int
	errs  []error // returned in order; past the end returns nil
}

func (f *fakeInvoker) Invoke(_ context.Context, _ bin.Encoder, _ bin.Decoder) error {
	i := f.calls
	f.calls++
	if i < len(f.errs) {
		return f.errs[i]
	}
	return nil
}

func TestFloodWaitMiddleware_failMode_returnsTypedError(t *testing.T) {
	inv := &fakeInvoker{errs: []error{tgerr.New(420, "FLOOD_WAIT_5")}}
	h := FloodWaitMiddleware(FloodFail, 0).Handle(inv)
	err := h.Invoke(context.Background(), nil, nil)
	require.ErrorIs(t, err, ErrFloodWait)
	require.Equal(t, 1, inv.calls, "fail mode must not retry")
}

func TestFloodWaitMiddleware_waitMode_retriesAfterFlood(t *testing.T) {
	// First call floods (1s, within cap), second succeeds.
	inv := &fakeInvoker{errs: []error{tgerr.New(420, "FLOOD_WAIT_1")}}
	h := FloodWaitMiddleware(FloodWait, 5).Handle(inv)
	start := time.Now()
	err := h.Invoke(context.Background(), nil, nil)
	require.NoError(t, err)
	require.Equal(t, 2, inv.calls, "wait mode must retry once after a flood")
	require.GreaterOrEqual(t, time.Since(start), 900*time.Millisecond)
}

func TestFloodWaitMiddleware_waitMode_overCapReturnsError(t *testing.T) {
	inv := &fakeInvoker{errs: []error{tgerr.New(420, "FLOOD_WAIT_60")}}
	h := FloodWaitMiddleware(FloodWait, 5).Handle(inv)
	err := h.Invoke(context.Background(), nil, nil)
	require.ErrorIs(t, err, ErrFloodWait)
	require.Equal(t, 1, inv.calls, "over-cap flood must not retry")
}

func TestFloodWaitMiddleware_nonFloodPassThrough(t *testing.T) {
	boom := errors.New("boom")
	inv := &fakeInvoker{errs: []error{boom}}
	h := FloodWaitMiddleware(FloodWait, 5).Handle(inv)
	err := h.Invoke(context.Background(), nil, nil)
	require.Equal(t, boom, err)
	require.Equal(t, 1, inv.calls, "non-flood error must not retry")
}

func TestFloodWaitMiddleware_success_noRetry(t *testing.T) {
	inv := &fakeInvoker{} // no errors
	h := FloodWaitMiddleware(FloodWait, 5).Handle(inv)
	require.NoError(t, h.Invoke(context.Background(), nil, nil))
	require.Equal(t, 1, inv.calls)
}
