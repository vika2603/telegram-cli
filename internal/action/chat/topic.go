package chat

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// TopicsRequest is the raw request for `tg chat topics`.
type TopicsRequest struct {
	RawRef string
	Q      string
	Limit  int
}

// TopicsQuery is the normalized payload passed to Telegram.
type TopicsQuery struct {
	Ref   ref.Ref
	Q     string
	Limit int
}

// TopicsFunc lists forum topics.
type TopicsFunc func(context.Context, TopicsQuery) ([]output.TopicRow, error)

// Topics validates and dispatches a topic-list request.
func Topics(ctx context.Context, req TopicsRequest, do TopicsFunc) ([]output.TopicRow, error) {
	q, err := NormalizeTopics(req)
	if err != nil {
		return nil, err
	}
	if do == nil {
		return nil, fmt.Errorf("%w: chat topics called without fetch function", command.ErrPrecondition)
	}
	return do(ctx, q)
}

// NormalizeTopics parses the ref and clamps the limit.
func NormalizeTopics(req TopicsRequest) (TopicsQuery, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return TopicsQuery{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	return TopicsQuery{Ref: parsed, Q: req.Q, Limit: limit}, nil
}

// CreateTopicRequest is the raw request for `tg chat topics create`.
type CreateTopicRequest struct {
	RawRef      string
	Title       string
	IconColor   int
	IconEmojiID int64
	RandomID    int64
}

// CreateTopicQuery is the normalized payload passed to Telegram.
type CreateTopicQuery struct {
	Ref         ref.Ref
	Title       string
	IconColor   int
	IconEmojiID int64
	RandomID    int64
}

// CreateTopicFunc creates a forum topic.
type CreateTopicFunc func(context.Context, CreateTopicQuery) (output.TopicRow, error)

// maxTopicTitleLen is Telegram's limit on a forum topic title (UTF-8 chars).
const maxTopicTitleLen = 128

// CreateTopic validates and dispatches a topic-create request.
func CreateTopic(ctx context.Context, req CreateTopicRequest, do CreateTopicFunc) (output.TopicRow, error) {
	q, err := NormalizeCreateTopic(req)
	if err != nil {
		return output.TopicRow{}, err
	}
	if do == nil {
		return output.TopicRow{}, fmt.Errorf("%w: chat topics create called without create function", command.ErrPrecondition)
	}
	return do(ctx, q)
}

// NormalizeCreateTopic parses the ref and validates the title.
func NormalizeCreateTopic(req CreateTopicRequest) (CreateTopicQuery, error) {
	if req.Title == "" {
		return CreateTopicQuery{}, fmt.Errorf("%w: topic title is required", command.ErrUsage)
	}
	if utf8.RuneCountInString(req.Title) > maxTopicTitleLen {
		return CreateTopicQuery{}, fmt.Errorf("%w: topic title exceeds %d characters", command.ErrUsage, maxTopicTitleLen)
	}
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return CreateTopicQuery{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	return CreateTopicQuery{
		Ref:         parsed,
		Title:       req.Title,
		IconColor:   req.IconColor,
		IconEmojiID: req.IconEmojiID,
		RandomID:    req.RandomID,
	}, nil
}
