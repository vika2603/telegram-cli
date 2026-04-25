package session

import (
	"context"
	"errors"
	"testing"
	"time"

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
