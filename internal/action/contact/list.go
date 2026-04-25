// Package contact contains contact-oriented command actions.
package contact

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

// ListRequest is the raw request for `tg contact list`.
type ListRequest struct {
	Blocked    bool
	MutualOnly bool
	Bots       bool
}

// ListQuery is the validated query passed to the Telegram data loader.
type ListQuery struct {
	Blocked    bool
	MutualOnly bool
	Bots       bool
}

// ListFunc loads contact rows after the request has been validated.
type ListFunc func(context.Context, ListQuery) ([]output.ContactRow, error)

// List validates the request and delegates data loading.
func List(ctx context.Context, req ListRequest, fetch ListFunc) ([]output.ContactRow, error) {
	if fetch == nil {
		return nil, fmt.Errorf("%w: contact list called without fetch function", command.ErrPrecondition)
	}
	return fetch(ctx, ListQuery{
		Blocked:    req.Blocked,
		MutualOnly: req.MutualOnly,
		Bots:       req.Bots,
	})
}
