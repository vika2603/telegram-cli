// Package password hosts the "tg password" command group (cloud 2FA).
package password

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/cli/password/disable"
	"github.com/vika2603/telegram-cli/internal/cli/password/set"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// New builds the cobra command group for "tg password".
func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "password",
		Short:   "Manage cloud (2FA) password",
		GroupID: "setup",
	}
	cmd.AddCommand(set.New(f, nil))
	cmd.AddCommand(disable.New(f, nil))
	return cmd
}
