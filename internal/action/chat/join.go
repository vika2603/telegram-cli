package chat

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// ---------------------------------------------------------------------------
// join list (pending join requests)
// ---------------------------------------------------------------------------

// JoinListRequest is the raw request for `tg chat join list`.
type JoinListRequest struct {
	RawRef string
	Link   string
	Limit  int
}

// JoinListQuery is the normalized payload passed to Telegram.
type JoinListQuery struct {
	Ref   ref.Ref
	Link  string
	Limit int
}

// JoinListFunc lists pending join requests.
type JoinListFunc func(context.Context, JoinListQuery) ([]output.MemberRow, error)

// JoinList validates and dispatches `tg chat join list`.
func JoinList(ctx context.Context, req JoinListRequest, do JoinListFunc) ([]output.MemberRow, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return nil, fmt.Errorf("%w: chat join list called without do function", command.ErrPrecondition)
	}
	return do(ctx, JoinListQuery{Ref: parsed, Link: req.Link, Limit: req.Limit})
}

// ---------------------------------------------------------------------------
// join approve / deny
// ---------------------------------------------------------------------------

// JoinDecisionRequest is the raw request for `tg chat join approve` / `deny`.
// Provide one or more users, or All to act on every pending request.
type JoinDecisionRequest struct {
	RawRef   string
	RawUsers []string
	All      bool
	Link     string
}

// JoinDecisionQuery is the normalized payload passed to Telegram.
type JoinDecisionQuery struct {
	Ref      ref.Ref
	Users    []ref.Ref
	All      bool
	Link     string
	Approved bool
}

// JoinDecisionFunc approves or rejects join requests; returns one row per user
// (or a single row for an --all decision).
type JoinDecisionFunc func(context.Context, JoinDecisionQuery) ([]output.JoinResultRow, error)

// ApproveJoin validates and dispatches `tg chat join approve`.
func ApproveJoin(ctx context.Context, req JoinDecisionRequest, do JoinDecisionFunc) ([]output.JoinResultRow, error) {
	return decideJoin(ctx, req, true, do)
}

// DenyJoin validates and dispatches `tg chat join deny`.
func DenyJoin(ctx context.Context, req JoinDecisionRequest, do JoinDecisionFunc) ([]output.JoinResultRow, error) {
	return decideJoin(ctx, req, false, do)
}

func decideJoin(ctx context.Context, req JoinDecisionRequest, approved bool, do JoinDecisionFunc) ([]output.JoinResultRow, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if req.All && len(req.RawUsers) > 0 {
		return nil, fmt.Errorf("%w: pass either user arguments or --all, not both", command.ErrUsage)
	}
	if !req.All && len(req.RawUsers) == 0 {
		return nil, fmt.Errorf("%w: provide one or more users, or --all", command.ErrUsage)
	}
	if do == nil {
		return nil, fmt.Errorf("%w: chat join decision called without do function", command.ErrPrecondition)
	}
	q := JoinDecisionQuery{Ref: parsed, All: req.All, Link: req.Link, Approved: approved}
	for _, raw := range req.RawUsers {
		u, err := ref.ParseRef(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid user ref %q: %s", command.ErrUsage, raw, err.Error())
		}
		q.Users = append(q.Users, u)
	}
	return do(ctx, q)
}
