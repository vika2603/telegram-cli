package chat

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// EditChatRequest is the raw request for `tg chat edit` / `tg channel edit`.
// Pointer fields change only when non-nil. Username == "" makes the chat
// private (removes the public @username); a non-empty value makes it public.
type EditChatRequest struct {
	RawRef   string
	Title    *string
	About    *string
	Username *string
}

// EditChatQuery is the normalized payload passed to Telegram.
type EditChatQuery struct {
	Ref      ref.Ref
	Title    *string
	About    *string
	Username *string
}

// EditChatFunc edits a supergroup/channel's title and/or about.
type EditChatFunc func(context.Context, EditChatQuery) (output.ChatRow, error)

// EditChat validates and dispatches a chat-edit request.
func EditChat(ctx context.Context, req EditChatRequest, do EditChatFunc) (output.ChatRow, error) {
	q, err := NormalizeEditChat(req)
	if err != nil {
		return output.ChatRow{}, err
	}
	if do == nil {
		return output.ChatRow{}, fmt.Errorf("%w: chat edit called without edit function", command.ErrPrecondition)
	}
	return do(ctx, q)
}

// NormalizeEditChat parses the ref and validates the requested changes.
func NormalizeEditChat(req EditChatRequest) (EditChatQuery, error) {
	if req.Title == nil && req.About == nil && req.Username == nil {
		return EditChatQuery{}, fmt.Errorf("%w: nothing to change; pass --title, --about, --public, or --private", command.ErrUsage)
	}
	if req.Title != nil {
		if *req.Title == "" {
			return EditChatQuery{}, fmt.Errorf("%w: --title cannot be empty", command.ErrUsage)
		}
		if utf8.RuneCountInString(*req.Title) > maxChatTitleLen {
			return EditChatQuery{}, fmt.Errorf("%w: title exceeds %d characters", command.ErrUsage, maxChatTitleLen)
		}
	}
	// Username "" is intentional (make private); a non-empty value must look
	// like a username — the server enforces the exact rules, we just catch
	// the obvious mistakes early.
	if req.Username != nil && *req.Username != "" {
		n := len(*req.Username)
		if n < 5 || n > 32 {
			return EditChatQuery{}, fmt.Errorf("%w: username must be 5-32 characters", command.ErrUsage)
		}
	}
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return EditChatQuery{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	return EditChatQuery{Ref: parsed, Title: req.Title, About: req.About, Username: req.Username}, nil
}
