package chat

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// ---------------------------------------------------------------------------
// Invite
// ---------------------------------------------------------------------------

// InviteRequest is the raw request for `tg chat invite`.
type InviteRequest struct {
	RawRef   string
	RawUsers []string
}

// InviteQuery is the normalized payload passed to the Telegram layer.
type InviteQuery struct {
	Ref   ref.Ref
	Users []ref.Ref
}

// InviteFunc invites users and returns one InviteRow per requested user.
type InviteFunc func(context.Context, InviteQuery) ([]output.InviteRow, error)

// Invite validates the request and dispatches the invite operation.
func Invite(ctx context.Context, req InviteRequest, do InviteFunc) ([]output.InviteRow, error) {
	if len(req.RawUsers) == 0 {
		return nil, fmt.Errorf("%w: at least one user argument is required", command.ErrUsage)
	}
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return nil, fmt.Errorf("%w: chat invite called without invite function", command.ErrPrecondition)
	}
	users := make([]ref.Ref, 0, len(req.RawUsers))
	for _, raw := range req.RawUsers {
		u, err := ref.ParseRef(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid user ref %q: %s", command.ErrUsage, raw, err.Error())
		}
		users = append(users, u)
	}
	return do(ctx, InviteQuery{Ref: parsed, Users: users})
}

// ---------------------------------------------------------------------------
// Ban / Unban
// ---------------------------------------------------------------------------

// BanRequest is the raw request for `tg chat ban` / `tg chat unban`.
type BanRequest struct {
	RawRef   string
	RawUser  string
	Unban    bool
	Yes      bool
	Prompter ui.Prompter
}

// BanQuery is the normalized payload passed to the Telegram layer.
type BanQuery struct {
	Ref   ref.Ref
	User  ref.Ref
	Unban bool
}

// BanFunc bans or unbans a user and returns the affected peer.
type BanFunc func(context.Context, BanQuery) (output.PeerRef, error)

// Ban validates the request, optionally confirms, and dispatches the operation.
func Ban(ctx context.Context, req BanRequest, do BanFunc) (output.PeerRef, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.PeerRef{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	userRef, err := ref.ParseRef(req.RawUser)
	if err != nil {
		return output.PeerRef{}, fmt.Errorf("%w: invalid user ref %q: %s", command.ErrUsage, req.RawUser, err.Error())
	}
	if do == nil {
		return output.PeerRef{}, fmt.Errorf("%w: chat ban called without ban function", command.ErrPrecondition)
	}
	if !req.Unban {
		msg := fmt.Sprintf("ban %s from %s?", userRef.String(), parsed.String())
		if err := ui.ConfirmDestructive(req.Prompter, msg, req.Yes); err != nil {
			return output.PeerRef{}, err
		}
	}
	return do(ctx, BanQuery{Ref: parsed, User: userRef, Unban: req.Unban})
}

// ---------------------------------------------------------------------------
// Promote / Demote
// ---------------------------------------------------------------------------

// PromoteRequest is the raw request for `tg chat promote` / `tg chat demote`.
type PromoteRequest struct {
	RawRef  string
	RawUser string
	Demote  bool
	Title   string // custom admin rank/title (promote only, <=16 chars)
}

// PromoteQuery is the normalized payload passed to the Telegram layer.
type PromoteQuery struct {
	Ref    ref.Ref
	User   ref.Ref
	Demote bool
	Title  string
}

// PromoteFunc promotes or demotes a user and returns the affected peer.
type PromoteFunc func(context.Context, PromoteQuery) (output.PeerRef, error)

// Promote validates the request and dispatches the promote/demote operation.
func Promote(ctx context.Context, req PromoteRequest, do PromoteFunc) (output.PeerRef, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.PeerRef{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	userRef, err := ref.ParseRef(req.RawUser)
	if err != nil {
		return output.PeerRef{}, fmt.Errorf("%w: invalid user ref %q: %s", command.ErrUsage, req.RawUser, err.Error())
	}
	if utf8.RuneCountInString(req.Title) > 16 {
		return output.PeerRef{}, fmt.Errorf("%w: --title must be at most 16 characters", command.ErrUsage)
	}
	if do == nil {
		return output.PeerRef{}, fmt.Errorf("%w: chat promote called without promote function", command.ErrPrecondition)
	}
	return do(ctx, PromoteQuery{Ref: parsed, User: userRef, Demote: req.Demote, Title: req.Title})
}
