// Package profile hosts the "tg profile" command group.
package profile

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/cli/profile/deletephoto"
	"github.com/vika2603/telegram-cli/internal/cli/profile/setbio"
	"github.com/vika2603/telegram-cli/internal/cli/profile/setname"
	"github.com/vika2603/telegram-cli/internal/cli/profile/setphoto"
	"github.com/vika2603/telegram-cli/internal/cli/profile/setstatus"
	"github.com/vika2603/telegram-cli/internal/cli/profile/setusername"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// New builds the cobra command group for "tg profile".
func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "profile",
		Short:   "Edit my Telegram profile (name, username, bio, photo, status)",
		GroupID: "core",
	}
	cmd.AddCommand(setname.New(f, nil))
	cmd.AddCommand(setusername.New(f, nil))
	cmd.AddCommand(setbio.New(f, nil))
	cmd.AddCommand(setphoto.New(f, nil))
	cmd.AddCommand(deletephoto.New(f, nil))
	cmd.AddCommand(setstatus.New(f, nil))
	return cmd
}
