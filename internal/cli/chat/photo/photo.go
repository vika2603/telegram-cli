// Package photo implements the "tg chat photo" command group
// (a group/channel's avatar).
package photo

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/runtime"
)

// New builds the "tg chat photo" command group.
func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "photo",
		Short: "Set or clear a group/channel photo",
	}
	cmd.AddCommand(newSet(f, nil))
	cmd.AddCommand(newClear(f, nil))
	return cmd
}
