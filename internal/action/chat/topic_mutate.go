package chat

import (
	"context"
	"fmt"
	"time"
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

// TopicInfoRequest is the raw request for `tg chat topics info`.
type TopicInfoRequest struct {
	RawRef  string
	TopicID int
}

// TopicInfoQuery is the normalized payload passed to Telegram.
type TopicInfoQuery struct {
	Ref     ref.Ref
	TopicID int
}

// TopicInfoFunc fetches a single forum topic.
type TopicInfoFunc func(context.Context, TopicInfoQuery) (output.TopicRow, error)

// InfoTopic validates and dispatches a topic-info request.
func InfoTopic(ctx context.Context, req TopicInfoRequest, do TopicInfoFunc) (output.TopicRow, error) {
	if req.TopicID <= 0 {
		return output.TopicRow{}, fmt.Errorf("%w: topic id must be positive", command.ErrUsage)
	}
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.TopicRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return output.TopicRow{}, fmt.Errorf("%w: chat topics info called without info function", command.ErrPrecondition)
	}
	return do(ctx, TopicInfoQuery{Ref: parsed, TopicID: req.TopicID})
}

// MuteTopicRequest is the raw request for `tg chat topics mute/unmute`. For
// mute, the target timestamp comes from Duration/Until/Forever (mutually
// exclusive); unmute ignores them and clears the mute.
type MuteTopicRequest struct {
	RawRef   string
	TopicID  int
	Unmute   bool
	Duration string
	Until    string
	Forever  bool
	Now      time.Time
}

// MuteTopicQuery is the normalized payload passed to Telegram.
type MuteTopicQuery struct {
	Ref       ref.Ref
	TopicID   int
	MuteUntil int
}

// MuteTopicFunc mutes or unmutes a forum topic.
type MuteTopicFunc func(context.Context, MuteTopicQuery) (output.TopicRow, error)

// MuteTopic validates and dispatches a topic mute/unmute request.
func MuteTopic(ctx context.Context, req MuteTopicRequest, do MuteTopicFunc) (output.TopicRow, error) {
	if req.TopicID <= 0 {
		return output.TopicRow{}, fmt.Errorf("%w: topic id must be positive", command.ErrUsage)
	}
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.TopicRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return output.TopicRow{}, fmt.Errorf("%w: chat topics mute called without mute function", command.ErrPrecondition)
	}
	muteUntil := 0
	if !req.Unmute {
		if err := command.MutuallyExclusive(
			"--duration, --until, and --forever are mutually exclusive",
			req.Duration != "", req.Until != "", req.Forever,
		); err != nil {
			return output.TopicRow{}, err
		}
		now := req.Now
		if now.IsZero() {
			now = time.Now()
		}
		secs, _, err := ResolveMuteUntil(MuteRequest{Duration: req.Duration, Until: req.Until, Forever: req.Forever}, now)
		if err != nil {
			return output.TopicRow{}, err
		}
		muteUntil = int(secs)
	}
	return do(ctx, MuteTopicQuery{Ref: parsed, TopicID: req.TopicID, MuteUntil: muteUntil})
}

// ReadTopicRequest is the raw request for `tg chat topics read`.
type ReadTopicRequest struct {
	RawRef  string
	TopicID int
}

// ReadTopicQuery is the normalized payload passed to Telegram.
type ReadTopicQuery struct {
	Ref     ref.Ref
	TopicID int
}

// ReadTopicFunc marks a forum topic as read.
type ReadTopicFunc func(context.Context, ReadTopicQuery) (output.TopicRow, error)

// ReadTopic validates and dispatches a topic-read request.
func ReadTopic(ctx context.Context, req ReadTopicRequest, do ReadTopicFunc) (output.TopicRow, error) {
	if req.TopicID <= 0 {
		return output.TopicRow{}, fmt.Errorf("%w: topic id must be positive", command.ErrUsage)
	}
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.TopicRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return output.TopicRow{}, fmt.Errorf("%w: chat topics read called without read function", command.ErrPrecondition)
	}
	return do(ctx, ReadTopicQuery{Ref: parsed, TopicID: req.TopicID})
}
