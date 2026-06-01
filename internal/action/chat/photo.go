package chat

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// PhotoRequest is the raw request for `tg chat photo set` / `clear`.
type PhotoRequest struct {
	RawRef string
	Path   string
}

// PhotoQuery is the normalized payload passed to the Telegram layer. Clear
// removes the current photo; otherwise Path is uploaded as the new one.
type PhotoQuery struct {
	Ref   ref.Ref
	Path  string
	Clear bool
}

// PhotoFunc sets or clears a group/channel photo.
type PhotoFunc func(context.Context, PhotoQuery) (output.ChatPhotoRow, error)

// SetPhoto validates `tg chat photo set` and dispatches.
func SetPhoto(ctx context.Context, req PhotoRequest, do PhotoFunc) (output.ChatPhotoRow, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.ChatPhotoRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if req.Path == "" {
		return output.ChatPhotoRow{}, fmt.Errorf("%w: a photo path is required (use - for stdin)", command.ErrUsage)
	}
	if do == nil {
		return output.ChatPhotoRow{}, fmt.Errorf("%w: chat photo set called without do function", command.ErrPrecondition)
	}
	return do(ctx, PhotoQuery{Ref: parsed, Path: req.Path})
}

// ClearPhoto validates `tg chat photo clear` and dispatches.
func ClearPhoto(ctx context.Context, req PhotoRequest, do PhotoFunc) (output.ChatPhotoRow, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.ChatPhotoRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return output.ChatPhotoRow{}, fmt.Errorf("%w: chat photo clear called without do function", command.ErrPrecondition)
	}
	return do(ctx, PhotoQuery{Ref: parsed, Clear: true})
}
