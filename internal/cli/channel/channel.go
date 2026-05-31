// Package channel hosts the "tg channel" command group for broadcast
// channels. Operations shared with groups (list, join, leave, mute, …) stay
// under "tg chat"; the channel-specific create/delete/edit commands reuse the
// chat implementations with channel-appropriate help and flags.
package channel

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/cli/channel/discussion"
	chatcreate "github.com/vika2603/telegram-cli/internal/cli/chat/create"
	chatdelete "github.com/vika2603/telegram-cli/internal/cli/chat/delete"
	chatedit "github.com/vika2603/telegram-cli/internal/cli/chat/edit"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "channel",
		Short:   "Broadcast channels",
		GroupID: "core",
	}
	cmd.AddCommand(chatcreate.NewWith(f, nil, "Create a broadcast channel", true))
	cmd.AddCommand(chatdelete.NewWith(f, nil, "Delete a channel (irreversible)"))
	cmd.AddCommand(chatedit.NewWith(f, nil, "Edit a channel: title, about, visibility, and settings", chatedit.ScopeChannel))
	cmd.AddCommand(discussion.New(f))
	return cmd
}
