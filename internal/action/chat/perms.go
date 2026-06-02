package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// RightKeys are the permission keywords accepted by --allow / --deny. Each
// maps (in the telegram layer) to one or more ChatBannedRights bits.
var RightKeys = map[string]bool{
	"send":     true, // text messages
	"media":    true, // photos/videos/files/audio/voice/round video
	"stickers": true, // stickers + GIFs
	"bots":     true, // inline bots + games
	"polls":    true,
	"links":    true, // link previews
	"invite":   true,
	"pin":      true,
	"info":     true, // change group info
	"topics":   true, // manage forum topics
}

func validateRightKeys(keys []string) error {
	for _, k := range keys {
		if !RightKeys[k] {
			return fmt.Errorf("%w: unknown permission %q (valid: send media stickers bots polls links invite pin info topics)", command.ErrUsage, k)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// restrict / unrestrict (per-user)
// ---------------------------------------------------------------------------

// RestrictRequest is the raw request for `tg chat restrict` / `unrestrict`.
type RestrictRequest struct {
	RawRef  string
	RawUser string
	Allow   []string
	Deny    []string
	Until   string // RFC3339 or duration (e.g. 1h, 7d); empty = permanent
	Now     time.Time
}

// RestrictQuery is the normalized payload passed to Telegram.
type RestrictQuery struct {
	Ref        ref.Ref
	User       ref.Ref
	Allow      []string
	Deny       []string
	UntilDate  int // unix seconds; 0 = permanent
	Unrestrict bool
}

// RestrictFunc restricts or unrestricts a single member.
type RestrictFunc func(context.Context, RestrictQuery) (output.RightsRow, error)

// Restrict validates and dispatches `tg chat restrict`.
func Restrict(ctx context.Context, req RestrictRequest, do RestrictFunc) (output.RightsRow, error) {
	q, err := normalizeRestrict(req)
	if err != nil {
		return output.RightsRow{}, err
	}
	if len(req.Allow) == 0 && len(req.Deny) == 0 {
		return output.RightsRow{}, fmt.Errorf("%w: pass --deny and/or --allow with permission keywords", command.ErrUsage)
	}
	if do == nil {
		return output.RightsRow{}, fmt.Errorf("%w: chat restrict called without do function", command.ErrPrecondition)
	}
	return do(ctx, q)
}

// Unrestrict validates and dispatches `tg chat unrestrict` (clears limits).
func Unrestrict(ctx context.Context, req RestrictRequest, do RestrictFunc) (output.RightsRow, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.RightsRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	user, err := ref.ParseRef(req.RawUser)
	if err != nil {
		return output.RightsRow{}, fmt.Errorf("%w: invalid user ref %q: %s", command.ErrUsage, req.RawUser, err.Error())
	}
	if do == nil {
		return output.RightsRow{}, fmt.Errorf("%w: chat unrestrict called without do function", command.ErrPrecondition)
	}
	return do(ctx, RestrictQuery{Ref: parsed, User: user, Unrestrict: true})
}

func normalizeRestrict(req RestrictRequest) (RestrictQuery, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return RestrictQuery{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	user, err := ref.ParseRef(req.RawUser)
	if err != nil {
		return RestrictQuery{}, fmt.Errorf("%w: invalid user ref %q: %s", command.ErrUsage, req.RawUser, err.Error())
	}
	if err := validateRightKeys(req.Allow); err != nil {
		return RestrictQuery{}, err
	}
	if err := validateRightKeys(req.Deny); err != nil {
		return RestrictQuery{}, err
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now()
	}
	until, err := parseExpire(req.Until, now)
	if err != nil {
		return RestrictQuery{}, err
	}
	return RestrictQuery{Ref: parsed, User: user, Allow: req.Allow, Deny: req.Deny, UntilDate: until}, nil
}

// ---------------------------------------------------------------------------
// perms (group default member rights)
// ---------------------------------------------------------------------------

// PermsRequest is the raw request for `tg chat perms`.
type PermsRequest struct {
	RawRef string
	Allow  []string
	Deny   []string
}

// PermsQuery is the normalized payload passed to Telegram.
type PermsQuery struct {
	Ref   ref.Ref
	Allow []string
	Deny  []string
}

// PermsFunc sets the group's default member permissions.
type PermsFunc func(context.Context, PermsQuery) (output.RightsRow, error)

// Perms validates and dispatches `tg chat perms`.
func Perms(ctx context.Context, req PermsRequest, do PermsFunc) (output.RightsRow, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.RightsRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if len(req.Allow) == 0 && len(req.Deny) == 0 {
		return output.RightsRow{}, fmt.Errorf("%w: pass --deny and/or --allow with permission keywords", command.ErrUsage)
	}
	if err := validateRightKeys(req.Allow); err != nil {
		return output.RightsRow{}, err
	}
	if err := validateRightKeys(req.Deny); err != nil {
		return output.RightsRow{}, err
	}
	if do == nil {
		return output.RightsRow{}, fmt.Errorf("%w: chat perms called without do function", command.ErrPrecondition)
	}
	return do(ctx, PermsQuery{Ref: parsed, Allow: req.Allow, Deny: req.Deny})
}
