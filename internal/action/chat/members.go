package chat

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

const maxMembersLimit = 1000

// MembersRequest is the raw request for `tg chat member list`.
type MembersRequest struct {
	RawRef string
	Filter string
	Q      string
	Limit  int
}

// MembersQuery is the normalized request passed to the Telegram layer.
type MembersQuery struct {
	Ref    ref.Ref
	Filter string
	Q      string
	Limit  int
}

// MembersFunc loads member rows after validation.
type MembersFunc func(context.Context, MembersQuery) ([]output.MemberRow, error)

var validMemberFilters = map[string]bool{
	"recent":   true,
	"admins":   true,
	"bots":     true,
	"kicked":   true,
	"banned":   true,
	"contacts": true,
}

var memberFiltersAcceptingQ = map[string]bool{
	"kicked":   true,
	"banned":   true,
	"contacts": true,
}

// Members validates the request and delegates member loading.
func Members(ctx context.Context, req MembersRequest, fetch MembersFunc) ([]output.MemberRow, error) {
	if !validMemberFilters[req.Filter] {
		return nil, fmt.Errorf("%w: invalid --filter %q", command.ErrUsage, req.Filter)
	}
	if req.Q != "" && !memberFiltersAcceptingQ[req.Filter] {
		return nil, fmt.Errorf("%w: --q is incompatible with --filter %s", command.ErrUsage, req.Filter)
	}
	if req.Limit <= 0 {
		return nil, fmt.Errorf("%w: --limit must be positive", command.ErrUsage)
	}
	if fetch == nil {
		return nil, fmt.Errorf("%w: chat member list called without fetch function", command.ErrPrecondition)
	}
	if req.Limit > maxMembersLimit {
		req.Limit = maxMembersLimit
	}
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	return fetch(ctx, MembersQuery{
		Ref:    parsed,
		Filter: req.Filter,
		Q:      req.Q,
		Limit:  req.Limit,
	})
}
