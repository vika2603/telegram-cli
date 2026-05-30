// Package rename implements "tg auth rename <old> <new>".
package rename

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/account"
	actionauth "github.com/vika2603/telegram-cli/internal/action/auth"
	"github.com/vika2603/telegram-cli/internal/authflow"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/daemon"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// Options holds flag values and injectable dependencies for "auth rename".
type Options struct {
	Old      string
	New      string
	Exporter output.Exporter
	F        *runtime.Invocation
}

// New constructs the cobra.Command for "auth rename". When runF is nil,
// production logic (Run) is used; tests pass a capture lambda.
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{F: f}
	cmd := &cobra.Command{
		Use:               "rename <old> <new>",
		Short:             "Rename a local account slot",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: authflow.CompleteAccountNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Old = args[0]
			opts.New = args[1]
			if runF != nil {
				return runF(opts)
			}
			return Run(opts)
		},
	}
	command.SetMeta(cmd, command.Meta{SkipAuthCheck: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"name", "state", "api_id", "default"})
	return cmd
}

// Run renames the local slot and emits the renamed account DTO. It honors
// --config / TG_CONFIG via opts.F for both the default_account check and the
// rewrite.
func Run(opts *Options) error {
	cfgPath := opts.F.ConfigPath
	if cfgPath == "" {
		cfgPath = account.ConfigFile()
	}

	var currentDefault string
	if opts.F.Config != nil {
		if cfg, err := opts.F.Config(); err == nil && cfg != nil && cfg.DefaultAccount != nil {
			currentDefault = *cfg.DefaultAccount
		}
	}

	result, err := actionauth.Rename(actionauth.RenameRequest{Old: opts.Old, New: opts.New}, actionauth.RenameDeps{
		ReadMeta:        account.ReadMeta,
		Rename:          account.RenameAccount,
		DaemonInstalled: daemonInstalled,
		CurrentDefault:  currentDefault,
		SetDefault: func(name string) error {
			return config.SetDefaultAccount(cfgPath, name)
		},
	})
	if err != nil {
		return err
	}
	if result.WarnName != "" {
		command.WarnLoosePermsByName(opts.F.IOStreams.ErrOut, result.WarnName)
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.F.IOStreams, result.DTO)
	}
	_, _ = opts.F.IOStreams.Out.Write([]byte(result.DTO.Human() + "\n"))
	return nil
}

// daemonInstalled reports whether a host service is registered for the
// account. It queries the service manager (launchd/systemd) rather than
// probing the socket, so an installed-but-stopped daemon is also detected —
// renaming under any registration would orphan the service.
func daemonInstalled(name string) bool {
	mgr, err := daemon.NewManager(name)
	if err != nil {
		return false
	}
	st, err := mgr.Status()
	return err == nil && st != nil && st.Installed
}
