package chat

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// ---------------------------------------------------------------------------
// invite create
// ---------------------------------------------------------------------------

// InviteLinkCreateRequest is the raw request for `tg chat invite create`.
type InviteLinkCreateRequest struct {
	RawRef        string
	Title         string
	Expire        string // RFC3339 or a duration like "24h", "7d"; empty = never
	UsageLimit    int
	RequestNeeded bool
	Now           time.Time
}

// InviteLinkCreateQuery is the normalized payload passed to Telegram.
type InviteLinkCreateQuery struct {
	Ref           ref.Ref
	Title         string
	ExpireDate    int // unix seconds; 0 = never
	UsageLimit    int
	RequestNeeded bool
}

// InviteLinkCreateFunc creates an invite link.
type InviteLinkCreateFunc func(context.Context, InviteLinkCreateQuery) (output.InviteLinkRow, error)

// InviteLinkCreate validates and dispatches an invite-link creation.
func InviteLinkCreate(ctx context.Context, req InviteLinkCreateRequest, do InviteLinkCreateFunc) (output.InviteLinkRow, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.InviteLinkRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if req.UsageLimit < 0 {
		return output.InviteLinkRow{}, fmt.Errorf("%w: --usage-limit must be >= 0", command.ErrUsage)
	}
	if do == nil {
		return output.InviteLinkRow{}, fmt.Errorf("%w: chat invite create called without do function", command.ErrPrecondition)
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now()
	}
	expireDate, err := parseExpire(req.Expire, now)
	if err != nil {
		return output.InviteLinkRow{}, err
	}
	return do(ctx, InviteLinkCreateQuery{
		Ref:           parsed,
		Title:         req.Title,
		ExpireDate:    expireDate,
		UsageLimit:    req.UsageLimit,
		RequestNeeded: req.RequestNeeded,
	})
}

func parseExpire(s string, now time.Time) (int, error) {
	if s == "" {
		return 0, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		if !t.After(now) {
			return 0, fmt.Errorf("%w: --expire must be in the future", command.ErrUsage)
		}
		if t.Unix() > math.MaxInt32 {
			return 0, fmt.Errorf("%w: --expire exceeds the 2038 epoch limit", command.ErrUsage)
		}
		return int(t.Unix()), nil
	}
	d, err := ParseDurationWithDays(s)
	if err != nil {
		return 0, fmt.Errorf("%w: --expire must be RFC3339 or a duration (e.g. 24h, 7d): %s", command.ErrUsage, err.Error())
	}
	if d <= 0 {
		return 0, fmt.Errorf("%w: --expire duration must be positive", command.ErrUsage)
	}
	return int(now.Add(d).Unix()), nil
}

// ---------------------------------------------------------------------------
// invite list
// ---------------------------------------------------------------------------

// InviteLinkListRequest is the raw request for `tg chat invite list`.
type InviteLinkListRequest struct {
	RawRef   string
	RawAdmin string
	Revoked  bool
	Limit    int
}

// InviteLinkListQuery is the normalized payload passed to Telegram.
type InviteLinkListQuery struct {
	Ref     ref.Ref
	Admin   ref.Ref // zero value = self
	Revoked bool
	Limit   int
}

// InviteLinkListFunc lists invite links.
type InviteLinkListFunc func(context.Context, InviteLinkListQuery) ([]output.InviteLinkRow, error)

// InviteLinkList validates and dispatches an invite-link listing.
func InviteLinkList(ctx context.Context, req InviteLinkListRequest, do InviteLinkListFunc) ([]output.InviteLinkRow, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	q := InviteLinkListQuery{Ref: parsed, Revoked: req.Revoked, Limit: req.Limit}
	if req.RawAdmin != "" {
		admin, err := ref.ParseRef(req.RawAdmin)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid --admin ref %q: %s", command.ErrUsage, req.RawAdmin, err.Error())
		}
		q.Admin = admin
	}
	if do == nil {
		return nil, fmt.Errorf("%w: chat invite list called without do function", command.ErrPrecondition)
	}
	return do(ctx, q)
}

// ---------------------------------------------------------------------------
// invite revoke / delete
// ---------------------------------------------------------------------------

// InviteLinkRequest is the raw request for `tg chat invite revoke` / `delete`.
type InviteLinkRequest struct {
	RawRef string
	Link   string
}

// InviteLinkQuery is the normalized payload passed to Telegram.
type InviteLinkQuery struct {
	Ref  ref.Ref
	Link string
}

// InviteLinkFunc revokes or deletes a single invite link.
type InviteLinkFunc func(context.Context, InviteLinkQuery) (output.InviteLinkRow, error)

// InviteLinkRevoke validates and dispatches `tg chat invite revoke`.
func InviteLinkRevoke(ctx context.Context, req InviteLinkRequest, do InviteLinkFunc) (output.InviteLinkRow, error) {
	return inviteLinkOp(ctx, req, do, "revoke")
}

// InviteLinkDelete validates and dispatches `tg chat invite delete`.
func InviteLinkDelete(ctx context.Context, req InviteLinkRequest, do InviteLinkFunc) (output.InviteLinkRow, error) {
	return inviteLinkOp(ctx, req, do, "delete")
}

func inviteLinkOp(ctx context.Context, req InviteLinkRequest, do InviteLinkFunc, op string) (output.InviteLinkRow, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.InviteLinkRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if req.Link == "" {
		return output.InviteLinkRow{}, fmt.Errorf("%w: a link argument is required", command.ErrUsage)
	}
	if do == nil {
		return output.InviteLinkRow{}, fmt.Errorf("%w: chat invite %s called without do function", command.ErrPrecondition, op)
	}
	return do(ctx, InviteLinkQuery{Ref: parsed, Link: req.Link})
}
