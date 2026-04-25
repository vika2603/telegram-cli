package contact

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/ref"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// PeerRequest is the raw request for contact commands that target one peer.
type PeerRequest struct {
	RawRef   string
	Yes      bool
	Prompter ui.Prompter
}

// PeerQuery is the validated peer query passed to the Telegram mutation.
type PeerQuery struct {
	Ref ref.Ref
}

// PeerFunc mutates one contact peer after request validation.
type PeerFunc func(context.Context, PeerQuery) error

// Block validates a contact block request, confirms it, and delegates mutation.
func Block(ctx context.Context, req PeerRequest, block PeerFunc) error {
	query, err := normalizePeer(req.RawRef)
	if err != nil {
		return err
	}
	if block == nil {
		return fmt.Errorf("%w: contact block called without block function", command.ErrPrecondition)
	}
	if err := ui.ConfirmDestructive(req.Prompter, fmt.Sprintf("block %s?", req.RawRef), req.Yes); err != nil {
		return err
	}
	return block(ctx, query)
}

// Delete validates a contact delete request, confirms it, and delegates mutation.
func Delete(ctx context.Context, req PeerRequest, delete PeerFunc) error {
	query, err := normalizePeer(req.RawRef)
	if err != nil {
		return err
	}
	if delete == nil {
		return fmt.Errorf("%w: contact delete called without delete function", command.ErrPrecondition)
	}
	if err := ui.ConfirmDestructive(req.Prompter, fmt.Sprintf("delete contact %s?", req.RawRef), req.Yes); err != nil {
		return err
	}
	return delete(ctx, query)
}

// Unblock validates a contact unblock request and delegates mutation.
func Unblock(ctx context.Context, req PeerRequest, unblock PeerFunc) error {
	query, err := normalizePeer(req.RawRef)
	if err != nil {
		return err
	}
	if unblock == nil {
		return fmt.Errorf("%w: contact unblock called without unblock function", command.ErrPrecondition)
	}
	return unblock(ctx, query)
}

func normalizePeer(raw string) (PeerQuery, error) {
	parsed, err := ref.ParseRef(raw)
	if err != nil {
		return PeerQuery{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	return PeerQuery{Ref: parsed}, nil
}
