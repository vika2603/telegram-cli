// Package search contains search-oriented command actions.
package search

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

const maxChatLimit = 1000

var validChatKinds = map[string]bool{"": true, "user": true, "group": true, "channel": true, "bot": true}

// ChatRequest is the raw request for `tg search chat`.
type ChatRequest struct {
	Query  string
	Kind   string
	MyOnly bool
	Limit  int
}

// ChatQuery is the validated query passed to the Telegram data loader.
type ChatQuery struct {
	Query string
	Limit int
}

// ChatFunc loads search rows after the request has been validated.
type ChatFunc func(context.Context, ChatQuery) ([]output.SearchChatRow, error)

// Chat validates the request, delegates data loading, then applies local filters.
func Chat(ctx context.Context, req ChatRequest, fetch ChatFunc) ([]output.SearchChatRow, error) {
	if !validChatKinds[req.Kind] {
		return nil, fmt.Errorf("%w: invalid --kind %q", command.ErrUsage, req.Kind)
	}
	if req.Limit <= 0 {
		return nil, fmt.Errorf("%w: --limit must be positive", command.ErrUsage)
	}
	if req.Limit > maxChatLimit {
		req.Limit = maxChatLimit
	}
	if fetch == nil {
		return nil, fmt.Errorf("%w: search chat called without fetch function", command.ErrPrecondition)
	}

	raw, err := fetch(ctx, ChatQuery{Query: req.Query, Limit: req.Limit})
	if err != nil {
		return nil, err
	}
	out := make([]output.SearchChatRow, 0, len(raw))
	for _, r := range raw {
		if req.MyOnly && r.Source != "my" {
			continue
		}
		if !chatKindMatches(req.Kind, r.Kind) {
			continue
		}
		out = append(out, r)
		if len(out) >= req.Limit {
			break
		}
	}
	return out, nil
}

// chatKindMatches maps the user-facing --kind vocabulary ("group") to the
// internal ChatRow.Kind vocabulary ("chat", which covers both legacy groups
// and supergroups). Empty filter matches everything.
func chatKindMatches(filter, rowKind string) bool {
	if filter == "" {
		return true
	}
	if filter == "group" {
		return rowKind == "chat"
	}
	return filter == rowKind
}
