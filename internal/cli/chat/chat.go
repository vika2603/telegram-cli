// Package chat hosts the "tg chat" command group.
package chat

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/cli/chat/admin"
	"github.com/vika2603/telegram-cli/internal/cli/chat/archive"
	"github.com/vika2603/telegram-cli/internal/cli/chat/ban"
	"github.com/vika2603/telegram-cli/internal/cli/chat/create"
	chatdelete "github.com/vika2603/telegram-cli/internal/cli/chat/delete"
	"github.com/vika2603/telegram-cli/internal/cli/chat/edit"
	"github.com/vika2603/telegram-cli/internal/cli/chat/info"
	"github.com/vika2603/telegram-cli/internal/cli/chat/invite"
	"github.com/vika2603/telegram-cli/internal/cli/chat/join"
	"github.com/vika2603/telegram-cli/internal/cli/chat/leave"
	"github.com/vika2603/telegram-cli/internal/cli/chat/list"
	"github.com/vika2603/telegram-cli/internal/cli/chat/member"
	"github.com/vika2603/telegram-cli/internal/cli/chat/mute"
	"github.com/vika2603/telegram-cli/internal/cli/chat/pin"
	"github.com/vika2603/telegram-cli/internal/cli/chat/read"
	"github.com/vika2603/telegram-cli/internal/cli/chat/topic"
	"github.com/vika2603/telegram-cli/internal/cli/chat/unarchive"
	"github.com/vika2603/telegram-cli/internal/cli/chat/unmute"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "chat",
		Short:   "Dialogs, chats, and channels",
		GroupID: "core",
	}
	cmd.AddCommand(create.New(f, nil))
	cmd.AddCommand(chatdelete.New(f, nil))
	cmd.AddCommand(edit.New(f, nil))
	cmd.AddCommand(list.New(f, nil))
	cmd.AddCommand(info.New(f, nil))
	cmd.AddCommand(member.New(f))
	cmd.AddCommand(read.New(f, nil))
	cmd.AddCommand(join.New(f, nil))
	cmd.AddCommand(leave.New(f, nil))
	cmd.AddCommand(mute.New(f, nil))
	cmd.AddCommand(unmute.New(f, nil))
	cmd.AddCommand(archive.New(f, nil))
	cmd.AddCommand(unarchive.New(f, nil))
	cmd.AddCommand(pin.NewPin(f, nil))
	cmd.AddCommand(pin.NewUnpin(f, nil))
	cmd.AddCommand(topic.New(f))
	cmd.AddCommand(invite.New(f, nil))
	cmd.AddCommand(ban.NewBan(f, nil))
	cmd.AddCommand(ban.NewUnban(f, nil))
	cmd.AddCommand(admin.NewPromote(f, nil))
	cmd.AddCommand(admin.NewDemote(f, nil))
	return cmd
}
