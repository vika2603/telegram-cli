package chat

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// EditTopicRequest is the raw request for `tg chat topics edit`. Title/Closed/
// Hidden are pointers so "not passed" (nil) is distinct from "set to empty/
// false" — only non-nil fields are sent to editForumTopic.
type EditTopicRequest struct {
	RawRef  string
	TopicID int
	Title   *string
	Closed  *bool
	Hidden  *bool
}

// EditTopicQuery is the normalized payload passed to Telegram.
type EditTopicQuery struct {
	Ref     ref.Ref
	TopicID int
	Title   *string
	Closed  *bool
	Hidden  *bool
}

// EditTopicFunc edits a forum topic.
type EditTopicFunc func(context.Context, EditTopicQuery) (output.TopicRow, error)

// EditTopic validates and dispatches a topic-edit request.
func EditTopic(ctx context.Context, req EditTopicRequest, do EditTopicFunc) (output.TopicRow, error) {
	q, err := NormalizeEditTopic(req)
	if err != nil {
		return output.TopicRow{}, err
	}
	if do == nil {
		return output.TopicRow{}, fmt.Errorf("%w: chat topics edit called without edit function", command.ErrPrecondition)
	}
	return do(ctx, q)
}

// NormalizeEditTopic parses the ref, checks the topic id, and ensures at
// least one field changes.
func NormalizeEditTopic(req EditTopicRequest) (EditTopicQuery, error) {
	if req.TopicID <= 0 {
		return EditTopicQuery{}, fmt.Errorf("%w: topic id must be positive", command.ErrUsage)
	}
	if req.Title == nil && req.Closed == nil && req.Hidden == nil {
		return EditTopicQuery{}, fmt.Errorf("%w: nothing to change; pass --title, --close/--reopen, or --hide/--unhide", command.ErrUsage)
	}
	if req.Title != nil {
		if *req.Title == "" {
			return EditTopicQuery{}, fmt.Errorf("%w: --title cannot be empty", command.ErrUsage)
		}
		if utf8.RuneCountInString(*req.Title) > maxTopicTitleLen {
			return EditTopicQuery{}, fmt.Errorf("%w: topic title exceeds %d characters", command.ErrUsage, maxTopicTitleLen)
		}
	}
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return EditTopicQuery{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	return EditTopicQuery{Ref: parsed, TopicID: req.TopicID, Title: req.Title, Closed: req.Closed, Hidden: req.Hidden}, nil
}

// DeleteTopicRequest is the raw request for `tg chat topics delete`.
type DeleteTopicRequest struct {
	RawRef   string
	TopicID  int
	Yes      bool
	Prompter ui.Prompter
}

// DeleteTopicQuery is the normalized payload passed to Telegram.
type DeleteTopicQuery struct {
	Ref     ref.Ref
	TopicID int
}

// DeleteTopicFunc deletes a forum topic (and its history).
type DeleteTopicFunc func(context.Context, DeleteTopicQuery) error

// DeleteTopic validates, confirms, and dispatches a topic-delete request.
func DeleteTopic(ctx context.Context, req DeleteTopicRequest, do DeleteTopicFunc) (output.TopicRow, error) {
	if req.TopicID <= 0 {
		return output.TopicRow{}, fmt.Errorf("%w: topic id must be positive", command.ErrUsage)
	}
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.TopicRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return output.TopicRow{}, fmt.Errorf("%w: chat topics delete called without delete function", command.ErrPrecondition)
	}
	if err := ui.ConfirmDestructive(req.Prompter, fmt.Sprintf("delete topic %d in %s (this removes its messages)?", req.TopicID, parsed.String()), req.Yes); err != nil {
		return output.TopicRow{}, err
	}
	if err := do(ctx, DeleteTopicQuery{Ref: parsed, TopicID: req.TopicID}); err != nil {
		return output.TopicRow{}, err
	}
	return output.TopicRow{ID: req.TopicID}, nil
}

// PinTopicRequest is the raw request for `tg chat topics pin`.
type PinTopicRequest struct {
	RawRef  string
	TopicID int
	Unpin   bool
}

// PinTopicQuery is the normalized payload passed to Telegram.
type PinTopicQuery struct {
	Ref     ref.Ref
	TopicID int
	Pinned  bool
}

// PinTopicFunc pins or unpins a forum topic.
type PinTopicFunc func(context.Context, PinTopicQuery) (output.TopicRow, error)

// PinTopic validates and dispatches a pin/unpin request.
func PinTopic(ctx context.Context, req PinTopicRequest, do PinTopicFunc) (output.TopicRow, error) {
	if req.TopicID <= 0 {
		return output.TopicRow{}, fmt.Errorf("%w: topic id must be positive", command.ErrUsage)
	}
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.TopicRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return output.TopicRow{}, fmt.Errorf("%w: chat topics pin called without pin function", command.ErrPrecondition)
	}
	return do(ctx, PinTopicQuery{Ref: parsed, TopicID: req.TopicID, Pinned: !req.Unpin})
}
