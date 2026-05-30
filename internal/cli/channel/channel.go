// Package channel hosts the "tg channel" command group for broadcast
// channels. Operations shared with groups (list, join, leave, mute, …) stay
// under "tg chat"; only channel-specific create/delete live here.
package channel

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/cli/channel/create"
	channeldelete "github.com/vika2603/telegram-cli/internal/cli/channel/delete"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "channel",
		Short:   "Broadcast channels",
		GroupID: "core",
	}
	cmd.AddCommand(create.New(f, nil))
	cmd.AddCommand(channeldelete.New(f, nil))
	return cmd
}
