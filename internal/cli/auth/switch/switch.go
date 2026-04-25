// Package switchcmd implements "tg auth switch <name>". The package name
// differs from the directory because "switch" is a Go keyword. Import with:
//
//	import switchcmd "github.com/vika2603/telegram-cli/internal/cli/auth/switch"
package switchcmd

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/account"
	actionauth "github.com/vika2603/telegram-cli/internal/action/auth"
	"github.com/vika2603/telegram-cli/internal/authflow"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// Options holds flag values and injectable dependencies for "auth switch".
type Options struct {
	Name     string
	Exporter output.Exporter
	F        *runtime.Invocation
}

// New constructs the cobra.Command for "auth switch".
// When runF is nil, production logic (Run) is used.
// Tests pass a capture lambda to verify flag parsing without touching disk.
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{F: f}
	cmd := &cobra.Command{
		Use:               "switch <name>",
		Short:             "Set the default account for subsequent commands",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: authflow.CompleteAccountNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
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

// Run executes the "auth switch" logic: validates the name, ensures the
// account slot exists, writes the default_account setting to the active config
// file (honoring --config / TG_CONFIG via opts.F.ConfigPath), and emits the
// account DTO.
func Run(opts *Options) error {
	result, err := actionauth.Switch(actionauth.SwitchRequest{Name: opts.Name}, actionauth.SwitchDeps{
		ReadMeta: account.ReadMeta,
		SetDefault: func(name string) error {
			cfgPath := opts.F.ConfigPath
			if cfgPath == "" {
				cfgPath = account.ConfigFile()
			}
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
