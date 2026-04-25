// Package search contains search-oriented command actions.
package search

import (
	"context"
	"fmt"
	"time"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

const maxMessageLimit = 1000

// MessageFilter names the supported Telegram message search filters.
type MessageFilter string

const (
	MessageFilterPhotos     MessageFilter = "photos"
	MessageFilterVideo      MessageFilter = "video"
	MessageFilterDocument   MessageFilter = "document"
	MessageFilterVoice      MessageFilter = "voice"
	MessageFilterMusic      MessageFilter = "music"
	MessageFilterGIF        MessageFilter = "gif"
	MessageFilterURL        MessageFilter = "url"
	MessageFilterPinned     MessageFilter = "pinned"
	MessageFilterGeo        MessageFilter = "geo"
	MessageFilterMyMentions MessageFilter = "my-mentions"
	MessageFilterRoundVideo MessageFilter = "round-video"
	MessageFilterRoundVoice MessageFilter = "round-voice"
	MessageFilterPhoneCalls MessageFilter = "phone-calls"
	MessageFilterChatPhotos MessageFilter = "chat-photos"
	MessageFilterContacts   MessageFilter = "contacts"
	MessageFilterPoll       MessageFilter = "poll"
	MessageFilterPhotoVideo MessageFilter = "photo-video"
)

var validMessageFilters = map[MessageFilter]bool{
	"":                      true,
	MessageFilterPhotos:     true,
	MessageFilterVideo:      true,
	MessageFilterDocument:   true,
	MessageFilterVoice:      true,
	MessageFilterMusic:      true,
	MessageFilterGIF:        true,
	MessageFilterURL:        true,
	MessageFilterPinned:     true,
	MessageFilterGeo:        true,
	MessageFilterMyMentions: true,
	MessageFilterRoundVideo: true,
	MessageFilterRoundVoice: true,
	MessageFilterPhoneCalls: true,
	MessageFilterChatPhotos: true,
	MessageFilterContacts:   true,
	MessageFilterPoll:       true,
	MessageFilterPhotoVideo: true,
}

// MessageRequest is the raw request for `tg search msg`.
type MessageRequest struct {
	Query          string
	In             string
	Filter         string
	From           string
	MinDate        string
	MaxDate        string
	BroadcastsOnly bool
	GroupsOnly     bool
	UsersOnly      bool
	Missed         bool
	Limit          int
	Order          string
}

// MessageQuery is the validated query passed to the Telegram data loader.
type MessageQuery struct {
	Query          string
	InRef          *ref.Ref
	FromRef        *ref.Ref
	Filter         MessageFilter
	Missed         bool
	MinDate        time.Time
	MaxDate        time.Time
	BroadcastsOnly bool
	GroupsOnly     bool
	UsersOnly      bool
	Limit          int
	Asc            bool
}

// MessageFunc loads message search rows after the request has been validated.
type MessageFunc func(context.Context, MessageQuery) ([]output.SearchMsgRow, error)

// Message validates the request and delegates data loading.
func Message(ctx context.Context, req MessageRequest, fetch MessageFunc) ([]output.SearchMsgRow, error) {
	query, err := NormalizeMessage(req)
	if err != nil {
		return nil, err
	}
	if fetch == nil {
		return nil, fmt.Errorf("%w: search msg called without fetch function", command.ErrPrecondition)
	}
	return fetch(ctx, query)
}

// NormalizeMessage parses flags, refs, dates, ordering, and filters into a MessageQuery.
func NormalizeMessage(req MessageRequest) (MessageQuery, error) {
	filter := MessageFilter(req.Filter)
	if !validMessageFilters[filter] {
		return MessageQuery{}, fmt.Errorf("%w: invalid --filter %q", command.ErrUsage, req.Filter)
	}
	if req.Limit <= 0 {
		return MessageQuery{}, fmt.Errorf("%w: --limit must be positive", command.ErrUsage)
	}
	if req.Limit > maxMessageLimit {
		req.Limit = maxMessageLimit
	}
	if req.From != "" && req.In == "" {
		return MessageQuery{}, fmt.Errorf("%w: --from requires --in (sender filter is in-chat only)", command.ErrUsage)
	}
	switch req.Order {
	case "", "asc", "desc":
	default:
		return MessageQuery{}, fmt.Errorf("%w: --order must be asc or desc", command.ErrUsage)
	}
	if req.Missed && filter != MessageFilterPhoneCalls {
		return MessageQuery{}, fmt.Errorf("%w: --missed requires --filter phone-calls", command.ErrUsage)
	}
	originFilters := 0
	for _, enabled := range []bool{req.BroadcastsOnly, req.GroupsOnly, req.UsersOnly} {
		if enabled {
			originFilters++
		}
	}
	if originFilters > 1 {
		return MessageQuery{}, fmt.Errorf("%w: only one of --broadcasts-only, --groups-only, --users-only may be set", command.ErrUsage)
	}
	if req.In != "" && originFilters > 0 {
		return MessageQuery{}, fmt.Errorf("%w: origin filters are global-search only", command.ErrUsage)
	}

	query := MessageQuery{
		Query:          req.Query,
		Filter:         filter,
		Missed:         req.Missed,
		BroadcastsOnly: req.BroadcastsOnly,
		GroupsOnly:     req.GroupsOnly,
		UsersOnly:      req.UsersOnly,
		Limit:          req.Limit,
		Asc:            req.Order == "asc",
	}

	if req.In != "" {
		parsed, err := ref.ParseRef(req.In)
		if err != nil {
			return MessageQuery{}, fmt.Errorf("%w: --in: %s", command.ErrUsage, err.Error())
		}
		query.InRef = &parsed
	}
	if req.From != "" {
		parsed, err := ref.ParseRef(req.From)
		if err != nil {
			return MessageQuery{}, fmt.Errorf("%w: --from: %s", command.ErrUsage, err.Error())
		}
		query.FromRef = &parsed
	}
	if req.MinDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.MinDate)
		if err != nil {
			return MessageQuery{}, fmt.Errorf("%w: --min-date: %s", command.ErrUsage, err.Error())
		}
		query.MinDate = parsed
	}
	if req.MaxDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.MaxDate)
		if err != nil {
			return MessageQuery{}, fmt.Errorf("%w: --max-date: %s", command.ErrUsage, err.Error())
		}
		query.MaxDate = parsed
	}

	return query, nil
}
