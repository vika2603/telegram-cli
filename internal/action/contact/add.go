package contact

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

// AddRequest is the raw request for `tg contact add`.
type AddRequest struct {
	Phone  string
	First  string
	Last   string
	Mutual bool
}

// AddQuery is the validated query passed to the Telegram data loader.
type AddQuery struct {
	Phone  string
	First  string
	Last   string
	Mutual bool
}

// AddFunc adds one contact after the request has been validated.
type AddFunc func(context.Context, AddQuery) (output.ContactRow, error)

// Add validates the request and delegates the Telegram mutation.
func Add(ctx context.Context, req AddRequest, add AddFunc) (output.ContactRow, error) {
	if add == nil {
		return output.ContactRow{}, fmt.Errorf("%w: contact add called without add function", command.ErrPrecondition)
	}
	if req.Phone == "" {
		return output.ContactRow{}, fmt.Errorf("%w: phone is required", command.ErrUsage)
	}
	if req.First == "" {
		return output.ContactRow{}, fmt.Errorf("%w: --first is required", command.ErrUsage)
	}
	return add(ctx, AddQuery(req))
}
