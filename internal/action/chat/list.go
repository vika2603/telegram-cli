// Package chat contains chat-oriented command actions.
package chat

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

const maxListLimit = 1000

// ListRequest is the normalized request for `tg chat list`.
type ListRequest struct {
	Limit        int
	ArchivedOnly bool
	PinnedOnly   bool
}

// ListFunc loads chat rows after the request has been validated.
type ListFunc func(context.Context, ListRequest) ([]output.ChatRow, error)

// List validates the request and delegates data loading.
func List(ctx context.Context, req ListRequest, fetch ListFunc) ([]output.ChatRow, error) {
	if req.Limit <= 0 {
		return nil, fmt.Errorf("%w: --limit must be positive", command.ErrUsage)
	}
	if req.Limit > maxListLimit {
		req.Limit = maxListLimit
	}
	if fetch == nil {
		return nil, fmt.Errorf("%w: chat list called without fetch function", command.ErrPrecondition)
	}
	return fetch(ctx, req)
}
