package chat

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

// maxChatTitleLen is Telegram's limit on a channel/supergroup title.
const maxChatTitleLen = 128

// CreateChatRequest is the raw request for `tg chat create`.
type CreateChatRequest struct {
	Title     string
	About     string
	Broadcast bool
	Forum     bool
}

// CreateChatQuery is the normalized payload passed to Telegram.
type CreateChatQuery struct {
	Title     string
	About     string
	Broadcast bool
	Forum     bool
}

// CreateChatFunc creates a supergroup or channel.
type CreateChatFunc func(context.Context, CreateChatQuery) (output.ChatRow, error)

// CreateChat validates and dispatches a create request.
func CreateChat(ctx context.Context, req CreateChatRequest, do CreateChatFunc) (output.ChatRow, error) {
	q, err := NormalizeCreateChat(req)
	if err != nil {
		return output.ChatRow{}, err
	}
	if do == nil {
		return output.ChatRow{}, fmt.Errorf("%w: chat create called without create function", command.ErrPrecondition)
	}
	return do(ctx, q)
}

// NormalizeCreateChat validates the title and the type/forum combination.
func NormalizeCreateChat(req CreateChatRequest) (CreateChatQuery, error) {
	if req.Title == "" {
		return CreateChatQuery{}, fmt.Errorf("%w: title is required", command.ErrUsage)
	}
	if utf8.RuneCountInString(req.Title) > maxChatTitleLen {
		return CreateChatQuery{}, fmt.Errorf("%w: title exceeds %d characters", command.ErrUsage, maxChatTitleLen)
	}
	if req.Broadcast && req.Forum {
		return CreateChatQuery{}, fmt.Errorf("%w: --forum applies to supergroups, not --channel", command.ErrUsage)
	}
	return CreateChatQuery(req), nil
}
