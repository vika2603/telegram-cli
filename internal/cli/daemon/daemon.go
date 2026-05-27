// Package daemon implements the "tg daemon" command tree: per-account
// service install / uninstall / start / stop / status / logs, plus the
// hidden "run" worker entry the host service manager invokes.
package daemon

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/cli/daemon/install"
	"github.com/vika2603/telegram-cli/internal/cli/daemon/logs"
	daemonruncmd "github.com/vika2603/telegram-cli/internal/cli/daemon/run"
	daemonstartcmd "github.com/vika2603/telegram-cli/internal/cli/daemon/start"
	daemonstatuscmd "github.com/vika2603/telegram-cli/internal/cli/daemon/status"
	daemonstopcmd "github.com/vika2603/telegram-cli/internal/cli/daemon/stop"
	"github.com/vika2603/telegram-cli/internal/cli/daemon/uninstall"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// New builds the "tg daemon" parent command and registers all
// subcommands. The "run" subcommand is hidden — it is the worker entry
// invoked by the OS service definition and not meant for direct human use.
func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the per-account background service",
		Long: `Manage the per-account tg daemon. The daemon holds a long-lived
MTProto session and streams updates to ~/.config/tg/accounts/<account>/daemon/updates.ndjson,
which clients can tail. Use install/start/stop/status/logs to control it.

Service registration uses the host's native manager (launchd on macOS,
systemd-user on Linux). Each account name gets its own service
instance, so multiple accounts can run daemons in parallel.`,
	}
	cmd.AddCommand(install.New(f, nil))
	cmd.AddCommand(uninstall.New(f, nil))
	cmd.AddCommand(daemonstartcmd.New(f, nil))
	cmd.AddCommand(daemonstopcmd.New(f, nil))
	cmd.AddCommand(daemonstatuscmd.New(f, nil))
	cmd.AddCommand(logs.New(f, nil))
	cmd.AddCommand(daemonruncmd.New(f, nil))
	return cmd
}
