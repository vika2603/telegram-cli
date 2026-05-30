// Package auth hosts the "tg auth" command group — local slot management
// + login flow. Remote account operations (sessions, password, profile)
// live under their own top-level groups.
package auth

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/cli/auth/list"
	"github.com/vika2603/telegram-cli/internal/cli/auth/login"
	"github.com/vika2603/telegram-cli/internal/cli/auth/logout"
	"github.com/vika2603/telegram-cli/internal/cli/auth/rename"
	"github.com/vika2603/telegram-cli/internal/cli/auth/status"
	switchcmd "github.com/vika2603/telegram-cli/internal/cli/auth/switch"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "auth",
		Short:   "Manage local account slots and login flow",
		GroupID: "setup",
	}
	cmd.AddCommand(login.New(f, nil))
	cmd.AddCommand(logout.New(f, nil))
	cmd.AddCommand(list.New(f, nil))
	cmd.AddCommand(switchcmd.New(f, nil))
	cmd.AddCommand(status.New(f, nil))
	cmd.AddCommand(rename.New(f, nil))
	return cmd
}
