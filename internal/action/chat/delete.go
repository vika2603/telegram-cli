package chat

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// DeleteChatRequest is the raw request for `tg chat delete`.
type DeleteChatRequest struct {
	RawRef   string
	Revoke   bool
	Yes      bool
	Prompter ui.Prompter
}

// DeleteChatQuery is the normalized payload passed to Telegram.
type DeleteChatQuery struct {
	Ref    ref.Ref
	Revoke bool
}

// DeleteChatFunc deletes a channel/supergroup or removes a user DM.
type DeleteChatFunc func(context.Context, DeleteChatQuery) (output.PeerRef, error)

// DeleteChat validates, confirms, and dispatches a chat-delete request.
// Deleting a channel/supergroup is irreversible and affects every member;
// deleting a user DM removes the conversation from the account (and, with
// Revoke, the other side too). Either way it confirms unless Yes is set.
func DeleteChat(ctx context.Context, req DeleteChatRequest, do DeleteChatFunc) (output.PeerRef, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.PeerRef{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return output.PeerRef{}, fmt.Errorf("%w: chat delete called without delete function", command.ErrPrecondition)
	}
	prompt := fmt.Sprintf("delete %s? this cannot be undone", parsed.String())
	if req.Revoke {
		prompt = fmt.Sprintf("delete %s for everyone? this cannot be undone", parsed.String())
	}
	if err := ui.ConfirmDestructive(req.Prompter, prompt, req.Yes); err != nil {
		return output.PeerRef{}, err
	}
	return do(ctx, DeleteChatQuery{Ref: parsed, Revoke: req.Revoke})
}
