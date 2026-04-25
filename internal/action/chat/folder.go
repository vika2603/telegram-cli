package chat

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// FolderRequest is the raw request for archive/unarchive commands.
type FolderRequest struct {
	RawRef string
}

// FolderQuery is the normalized request passed to the Telegram layer.
type FolderQuery struct {
	Ref    ref.Ref
	Action string
	Folder int
}

// FolderFunc moves a chat to a target Telegram folder.
type FolderFunc func(context.Context, FolderQuery) (output.ChatFolderRow, error)

// Archive validates `tg chat archive` and delegates the Telegram call.
func Archive(ctx context.Context, req FolderRequest, do FolderFunc) (output.ChatFolderRow, error) {
	return moveFolder(ctx, req, "archive", 1, do)
}

// Unarchive validates `tg chat unarchive` and delegates the Telegram call.
func Unarchive(ctx context.Context, req FolderRequest, do FolderFunc) (output.ChatFolderRow, error) {
	return moveFolder(ctx, req, "unarchive", 0, do)
}

func moveFolder(ctx context.Context, req FolderRequest, action string, folder int, do FolderFunc) (output.ChatFolderRow, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return output.ChatFolderRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if do == nil {
		return output.ChatFolderRow{}, fmt.Errorf("%w: chat %s called without do function", command.ErrPrecondition, action)
	}
	return do(ctx, FolderQuery{Ref: parsed, Action: action, Folder: folder})
}
