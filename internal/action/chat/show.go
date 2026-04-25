package chat

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// ShowRequest is the raw request for `tg chat show`.
type ShowRequest struct {
	RawRef string
}

// ShowQuery is the validated query passed to the Telegram data loader.
type ShowQuery struct {
	Ref ref.Ref
}

// ShowFunc loads one chat row after the request has been validated.
type ShowFunc func(context.Context, ShowQuery) (output.ChatRow, error)

// Show validates the request and delegates data loading.
func Show(ctx context.Context, req ShowRequest, fetch ShowFunc) (output.ChatRow, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.ChatRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if fetch == nil {
		return output.ChatRow{}, fmt.Errorf("%w: chat show called without fetch function", command.ErrPrecondition)
	}
	return fetch(ctx, ShowQuery{Ref: parsed})
}
