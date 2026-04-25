package chat

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// MembershipRequest is the raw request for join/leave commands.
type MembershipRequest struct {
	RawRef   string
	Yes      bool
	Prompter ui.Prompter
}

// MembershipQuery is the normalized request passed to the Telegram layer.
type MembershipQuery struct {
	Ref ref.Ref
}

// MembershipFunc changes chat membership after validation.
type MembershipFunc func(context.Context, MembershipQuery) (output.ChatMembershipRow, error)

// Join validates `tg chat join` and delegates.
func Join(ctx context.Context, req MembershipRequest, do MembershipFunc) (output.ChatMembershipRow, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.ChatMembershipRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return output.ChatMembershipRow{}, fmt.Errorf("%w: chat join called without do function", command.ErrPrecondition)
	}
	return do(ctx, MembershipQuery{Ref: parsed})
}

// Leave confirms, validates `tg chat leave`, and delegates.
func Leave(ctx context.Context, req MembershipRequest, do MembershipFunc) (output.ChatMembershipRow, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.ChatMembershipRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return output.ChatMembershipRow{}, fmt.Errorf("%w: chat leave called without do function", command.ErrPrecondition)
	}
	if err := ui.ConfirmDestructive(req.Prompter, fmt.Sprintf("Leave %s?", req.RawRef), req.Yes); err != nil {
		return output.ChatMembershipRow{}, err
	}
	return do(ctx, MembershipQuery{Ref: parsed})
}
