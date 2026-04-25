// Package session hosts the "tg sessions" command group (remote authorizations).
package session

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/cli/session/list"
	"github.com/vika2603/telegram-cli/internal/cli/session/terminate"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// New builds the cobra command group for "tg sessions".
func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sessions",
		Short:   "Manage remote sessions",
		GroupID: "setup",
	}
	cmd.AddCommand(list.New(f, nil))
	cmd.AddCommand(terminate.New(f, nil))
	return cmd
}
