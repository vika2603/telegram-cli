package chat

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// ReadRequest is the raw request for `tg chat read`.
type ReadRequest struct {
	RawRef string
	MaxID  int
}

// ReadQuery is the normalized request passed to the Telegram layer.
type ReadQuery struct {
	Ref   ref.Ref
	MaxID int
}

// ReadFunc marks a chat as read after validation.
type ReadFunc func(context.Context, ReadQuery) error

// Read validates the request and delegates.
func Read(ctx context.Context, req ReadRequest, do ReadFunc) error {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return fmt.Errorf("%w: chat read called without do function", command.ErrPrecondition)
	}
	return do(ctx, ReadQuery{Ref: parsed, MaxID: req.MaxID})
}
