package chat

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// DiscussionRequest is the raw request for `tg channel discussion link/unlink`.
// RawGroup is empty when unlinking.
type DiscussionRequest struct {
	RawChannel string
	RawGroup   string
}

// DiscussionQuery is the normalized payload passed to the Telegram layer.
// Unlink discriminates direction: true clears the channel's discussion group,
// false links Group as the channel's discussion group.
type DiscussionQuery struct {
	Channel ref.Ref
	Group   ref.Ref
	Unlink  bool
}

// DiscussionFunc links or unlinks a channel's discussion group.
type DiscussionFunc func(context.Context, DiscussionQuery) (output.DiscussionRow, error)

// LinkDiscussion validates `tg channel discussion link` and dispatches.
func LinkDiscussion(ctx context.Context, req DiscussionRequest, do DiscussionFunc) (output.DiscussionRow, error) {
	channel, err := ref.ParseRef(req.RawChannel)
	if err != nil {
		return output.DiscussionRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	group, err := ref.ParseRef(req.RawGroup)
	if err != nil {
		return output.DiscussionRow{}, fmt.Errorf("%w: invalid group ref %q: %s", command.ErrUsage, req.RawGroup, err.Error())
	}
	if do == nil {
		return output.DiscussionRow{}, fmt.Errorf("%w: channel discussion link called without do function", command.ErrPrecondition)
	}
	return do(ctx, DiscussionQuery{Channel: channel, Group: group})
}

// UnlinkDiscussion validates `tg channel discussion unlink` and dispatches.
func UnlinkDiscussion(ctx context.Context, req DiscussionRequest, do DiscussionFunc) (output.DiscussionRow, error) {
	channel, err := ref.ParseRef(req.RawChannel)
	if err != nil {
		return output.DiscussionRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return output.DiscussionRow{}, fmt.Errorf("%w: channel discussion unlink called without do function", command.ErrPrecondition)
	}
	return do(ctx, DiscussionQuery{Channel: channel, Unlink: true})
}

// DiscussionCandidatesFunc lists supergroups eligible as a discussion group.
type DiscussionCandidatesFunc func(context.Context) ([]output.ChatRow, error)

// DiscussionCandidates dispatches `tg channel discussion candidates`.
func DiscussionCandidates(ctx context.Context, do DiscussionCandidatesFunc) ([]output.ChatRow, error) {
	if do == nil {
		return nil, fmt.Errorf("%w: channel discussion candidates called without do function", command.ErrPrecondition)
	}
	return do(ctx)
}
