package chat

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// MuteRequest is the raw request for `tg chat mute`.
type MuteRequest struct {
	RawRef   string
	Duration string
	Until    string
	Forever  bool
	Now      time.Time
}

// MuteQuery is the normalized request passed to the Telegram layer.
type MuteQuery struct {
	Ref       ref.Ref
	MuteUntil int32
}

// MuteFunc updates notification settings after validation.
type MuteFunc func(context.Context, MuteQuery) (output.ChatMuteRow, error)

// Mute validates mute flags, normalizes the target timestamp, and delegates.
func Mute(ctx context.Context, req MuteRequest, do MuteFunc) (output.ChatMuteRow, error) {
	if err := command.MutuallyExclusive(
		"--duration, --until, and --forever are mutually exclusive",
		req.Duration != "", req.Until != "", req.Forever,
	); err != nil {
		return output.ChatMuteRow{}, err
	}
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.ChatMuteRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return output.ChatMuteRow{}, fmt.Errorf("%w: chat mute called without do function", command.ErrPrecondition)
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now()
	}
	muteUntil, label, err := ResolveMuteUntil(req, now)
	if err != nil {
		return output.ChatMuteRow{}, err
	}
	row, err := do(ctx, MuteQuery{Ref: parsed, MuteUntil: muteUntil})
	if err != nil {
		return output.ChatMuteRow{}, err
	}
	if row.MuteUntil == "" {
		row.MuteUntil = label
	}
	return row, nil
}

// UnmuteRequest is the raw request for `tg chat unmute`.
type UnmuteRequest struct {
	RawRef string
}

// UnmuteQuery is the normalized request passed to the Telegram layer.
type UnmuteQuery struct {
	Ref ref.Ref
}

// UnmuteFunc restores notification settings after validation.
type UnmuteFunc func(context.Context, UnmuteQuery) (output.ChatMuteRow, error)

// Unmute validates the request and delegates.
func Unmute(ctx context.Context, req UnmuteRequest, do UnmuteFunc) (output.ChatMuteRow, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.ChatMuteRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return output.ChatMuteRow{}, fmt.Errorf("%w: chat unmute called without do function", command.ErrPrecondition)
	}
	return do(ctx, UnmuteQuery{Ref: parsed})
}

// ResolveMuteUntil turns flag values into a Unix-seconds int32 and a human label.
func ResolveMuteUntil(req MuteRequest, now time.Time) (int32, string, error) {
	switch {
	case req.Until != "":
		t, err := time.Parse(time.RFC3339, req.Until)
		if err != nil {
			return 0, "", fmt.Errorf("%w: --until must be RFC3339: %s", command.ErrUsage, err.Error())
		}
		if !t.After(now) {
			return 0, "", fmt.Errorf("%w: --until must be in the future", command.ErrUsage)
		}
		if t.Unix() > math.MaxInt32 {
			return 0, "", fmt.Errorf("%w: --until exceeds 2038-01-19 (int32 unix epoch limit); use --forever instead", command.ErrUsage)
		}
		return int32(t.Unix()), t.UTC().Format(time.RFC3339), nil
	case req.Duration != "":
		d, err := ParseDurationWithDays(req.Duration)
		if err != nil {
			return 0, "", fmt.Errorf("%w: --duration: %s", command.ErrUsage, err.Error())
		}
		if d <= 0 {
			return 0, "", fmt.Errorf("%w: --duration must be positive", command.ErrUsage)
		}
		t := now.Add(d)
		if t.Unix() > math.MaxInt32 {
			return 0, "", fmt.Errorf("%w: --duration pushes past 2038-01-19; use --forever instead", command.ErrUsage)
		}
		return int32(t.Unix()), t.UTC().Format(time.RFC3339), nil
	default:
		return math.MaxInt32, "forever", nil
	}
}

// ParseDurationWithDays parses Go durations plus an extra "d" suffix for days.
func ParseDurationWithDays(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "d") {
		nStr := strings.TrimSuffix(s, "d")
		n, err := strconv.ParseFloat(nStr, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot parse %q: %s", s, err.Error())
		}
		return time.Duration(n * float64(24*time.Hour)), nil
	}
	return time.ParseDuration(s)
}
