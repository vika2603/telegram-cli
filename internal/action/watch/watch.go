// Package watch contains the action layer for "tg watch".
package watch

import (
	"context"
	"fmt"
	"strings"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// Request is the unvalidated input from CLI flags.
type Request struct {
	// RawRefs is zero-or-more peer refs (@username, c:CHANNEL_ID, etc.).
	// Empty means "all dialogs".
	RawRefs []string
	// Kinds restricts which event kinds to emit: "message", "edit", "delete".
	// Empty means all.
	Kinds []string
	// Limit caps the number of events emitted before Watch returns. 0 means
	// stream until ctx is cancelled (typically Ctrl-C).
	Limit int
}

// Query is the validated form of Request handed to the stream loop.
type Query struct {
	Refs  []ref.Ref
	Kinds []string
	Limit int
}

// StreamFunc is the injectable seam Run calls to actually subscribe and
// stream events. Production wiring lives in internal/cli/watch.
type StreamFunc func(context.Context, Query) error

// Normalize parses each RawRef into a ref.Ref and validates the kind set.
func Normalize(req Request) (Query, error) {
	q := Query{Limit: req.Limit}
	if req.Limit < 0 {
		return q, fmt.Errorf("%w: --limit must be >= 0", command.ErrUsage)
	}
	for _, raw := range req.RawRefs {
		r, err := ref.ParseRef(raw)
		if err != nil {
			return q, fmt.Errorf("%w: %w", command.ErrUsage, err)
		}
		q.Refs = append(q.Refs, r)
	}
	allowedKinds := map[string]struct{}{"message": {}, "edit": {}, "delete": {}}
	for _, k := range req.Kinds {
		k = strings.TrimSpace(strings.ToLower(k))
		if k == "" {
			continue
		}
		if _, ok := allowedKinds[k]; !ok {
			return q, fmt.Errorf("%w: unknown event kind %q (valid: message, edit, delete)", command.ErrUsage, k)
		}
		q.Kinds = append(q.Kinds, k)
	}
	return q, nil
}

// Run validates the request and delegates streaming to stream.
func Run(ctx context.Context, req Request, stream StreamFunc) error {
	q, err := Normalize(req)
	if err != nil {
		return err
	}
	if stream == nil {
		return fmt.Errorf("%w: watch called without stream function", command.ErrPrecondition)
	}
	return stream(ctx, q)
}
