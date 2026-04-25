// Package message contains message-oriented command actions.
package message

import (
	"context"
	"fmt"
	"time"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

const maxListLimit = 1000

// ListRequest is the raw request for `tg msg list`.
type ListRequest struct {
	RawRef  string
	Limit   int
	MinDate string
	MaxDate string
	Order   string
}

// ListQuery is the validated query passed to the Telegram data loader.
type ListQuery struct {
	Ref     ref.Ref
	Limit   int
	MinDate time.Time
	MaxDate time.Time
	Asc     bool
}

// ListFunc loads message rows after the request has been validated.
type ListFunc func(context.Context, ListQuery) ([]output.MessageRow, error)

// List validates the request and delegates data loading.
func List(ctx context.Context, req ListRequest, fetch ListFunc) ([]output.MessageRow, error) {
	query, err := NormalizeList(req)
	if err != nil {
		return nil, err
	}
	if fetch == nil {
		return nil, fmt.Errorf("%w: msg list called without fetch function", command.ErrPrecondition)
	}
	return fetch(ctx, query)
}

// NormalizeList parses flags and refs into a ListQuery.
func NormalizeList(req ListRequest) (ListQuery, error) {
	if req.Limit <= 0 {
		return ListQuery{}, fmt.Errorf("%w: --limit must be positive", command.ErrUsage)
	}
	if req.Limit > maxListLimit {
		req.Limit = maxListLimit
	}
	switch req.Order {
	case "", "asc", "desc":
	default:
		return ListQuery{}, fmt.Errorf("%w: --order must be asc or desc", command.ErrUsage)
	}

	var minT, maxT time.Time
	var err error
	if req.MinDate != "" {
		minT, err = time.Parse(time.RFC3339, req.MinDate)
		if err != nil {
			return ListQuery{}, fmt.Errorf("%w: --min-date: %s", command.ErrUsage, err.Error())
		}
	}
	if req.MaxDate != "" {
		maxT, err = time.Parse(time.RFC3339, req.MaxDate)
		if err != nil {
			return ListQuery{}, fmt.Errorf("%w: --max-date: %s", command.ErrUsage, err.Error())
		}
	}

	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return ListQuery{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	return ListQuery{
		Ref:     parsed,
		Limit:   req.Limit,
		MinDate: minT,
		MaxDate: maxT,
		Asc:     req.Order == "asc",
	}, nil
}
