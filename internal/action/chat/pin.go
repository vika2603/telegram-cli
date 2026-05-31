package chat

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// PinRequest is the raw request for `tg chat pin` / `tg chat unpin`.
type PinRequest struct {
	RawRef string
}

// PinQuery is the normalized request passed to the Telegram layer. Pinned
// discriminates direction: true pins the dialog, false unpins it.
type PinQuery struct {
	Ref    ref.Ref
	Pinned bool
}

// PinFunc toggles whether a dialog is pinned to the top of the chat list.
type PinFunc func(context.Context, PinQuery) (output.ChatPinRow, error)

// Pin validates `tg chat pin` and delegates the Telegram call.
func Pin(ctx context.Context, req PinRequest, do PinFunc) (output.ChatPinRow, error) {
	return togglePin(ctx, req, true, do)
}

// Unpin validates `tg chat unpin` and delegates the Telegram call.
func Unpin(ctx context.Context, req PinRequest, do PinFunc) (output.ChatPinRow, error) {
	return togglePin(ctx, req, false, do)
}

func togglePin(ctx context.Context, req PinRequest, pinned bool, do PinFunc) (output.ChatPinRow, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.ChatPinRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return output.ChatPinRow{}, fmt.Errorf("%w: chat pin called without do function", command.ErrPrecondition)
	}
	return do(ctx, PinQuery{Ref: parsed, Pinned: pinned})
}
