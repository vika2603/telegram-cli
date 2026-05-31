// Package discussion implements the "tg channel discussion" command group
// (a channel's linked discussion supergroup / comments).
package discussion

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/runtime"
)

// New builds the "tg channel discussion" command group.
func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discussion",
		Short: "Manage a channel's linked discussion group (comments)",
	}
	cmd.AddCommand(newLink(f, nil))
	cmd.AddCommand(newUnlink(f, nil))
	cmd.AddCommand(newCandidates(f, nil))
	return cmd
}
