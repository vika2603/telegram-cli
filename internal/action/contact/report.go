package contact

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/ref"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// ReportReasons are the keywords accepted by `tg contact report --reason`.
// Keep this set in sync with reportReason() in internal/telegram/contacts.go,
// which maps each keyword to a tg.ReportReasonClass — a keyword present here but
// missing there would silently report as spam.
var ReportReasons = map[string]bool{
	"spam":             true,
	"violence":         true,
	"porn":             true,
	"child-abuse":      true,
	"copyright":        true,
	"fake":             true,
	"drugs":            true,
	"personal-details": true,
	"geo-irrelevant":   true,
	"other":            true,
}

// ReportRequest is the raw request for `tg contact report`.
type ReportRequest struct {
	RawRef   string
	Reason   string
	Message  string
	Block    bool // also block the peer after reporting
	Yes      bool
	Prompter ui.Prompter
}

// ReportQuery is the validated payload passed to the Telegram layer.
type ReportQuery struct {
	Ref     ref.Ref
	Reason  string
	Message string
	Block   bool
}

// ReportFunc reports one peer after request validation.
type ReportFunc func(context.Context, ReportQuery) error

// Report validates a report request, confirms it, and delegates the call.
func Report(ctx context.Context, req ReportRequest, do ReportFunc) error {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	reason := req.Reason
	if reason == "" {
		reason = "spam"
	}
	if !ReportReasons[reason] {
		return fmt.Errorf("%w: unknown report reason %q (valid: spam violence porn child-abuse copyright fake drugs personal-details geo-irrelevant other)", command.ErrUsage, reason)
	}
	if do == nil {
		return fmt.Errorf("%w: contact report called without report function", command.ErrPrecondition)
	}
	prompt := fmt.Sprintf("report %s for %s?", req.RawRef, reason)
	if req.Block {
		prompt = fmt.Sprintf("report %s for %s and block them?", req.RawRef, reason)
	}
	if err := ui.ConfirmDestructive(req.Prompter, prompt, req.Yes); err != nil {
		return err
	}
	return do(ctx, ReportQuery{Ref: parsed, Reason: reason, Message: req.Message, Block: req.Block})
}
