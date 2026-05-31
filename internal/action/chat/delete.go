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
	Yes      bool
	Prompter ui.Prompter
}

// DeleteChatQuery is the normalized payload passed to Telegram.
type DeleteChatQuery struct {
	Ref ref.Ref
}

// DeleteChatFunc deletes a supergroup or channel.
type DeleteChatFunc func(context.Context, DeleteChatQuery) (output.PeerRef, error)

// DeleteChat validates, confirms, and dispatches a chat-delete request.
// Deleting a channel/supergroup is irreversible and affects every member, so
// it always confirms unless Yes is set.
func DeleteChat(ctx context.Context, req DeleteChatRequest, do DeleteChatFunc) (output.PeerRef, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.PeerRef{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return output.PeerRef{}, fmt.Errorf("%w: chat delete called without delete function", command.ErrPrecondition)
	}
	if err := ui.ConfirmDestructive(req.Prompter, fmt.Sprintf("delete %s for everyone? this cannot be undone", parsed.String()), req.Yes); err != nil {
		return output.PeerRef{}, err
	}
	return do(ctx, DeleteChatQuery{Ref: parsed})
}
