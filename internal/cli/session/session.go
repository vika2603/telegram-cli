// Package session hosts the "tg session" command group (remote authorizations).
package session

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/cli/session/list"
	"github.com/vika2603/telegram-cli/internal/cli/session/revoke"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// New builds the cobra command group for "tg session".
func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "session",
		Short:   "Manage remote sessions",
		GroupID: "setup",
	}
	cmd.AddCommand(list.New(f, nil))
	cmd.AddCommand(revoke.New(f, nil))
	return cmd
}
