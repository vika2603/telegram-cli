package member

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/cli/chat/setperms"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// New builds the "tg chat member" command group.
func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member",
		Short: "Members of a group or channel",
	}
	cmd.AddCommand(NewList(f, nil))
	cmd.AddCommand(setperms.NewSetPerms(f, nil))
	cmd.AddCommand(setperms.NewUnsetPerms(f, nil))
	return cmd
}
