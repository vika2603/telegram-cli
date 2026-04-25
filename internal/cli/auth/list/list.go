// Package list implements the "tg auth list" command.
package list

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/account"
	actionauth "github.com/vika2603/telegram-cli/internal/action/auth"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// Options holds both flag values and injectable dependencies for "account list".
type Options struct {
	Exporter output.Exporter
}

// New constructs the cobra.Command for "account list".
// When runF is nil, production logic (listRun) is used.
// Tests pass a capture lambda to verify flag parsing without touching disk.
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}

	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List accounts",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return listRun(f, opts)
		},
	}

	command.SetMeta(cmd, command.Meta{SkipAuthCheck: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"name", "state", "api_id", "default"})

	return cmd
}

func listRun(f *runtime.Invocation, opts *Options) error {
	result, err := actionauth.List(actionauth.ListDeps{
		Config:       f.Config,
		ListAccounts: account.ListAccounts,
		ReadMeta:     account.ReadMeta,
	})
	if err != nil {
		return err
	}

	for _, name := range result.WarnNames {
		command.WarnLoosePermsByName(f.IOStreams.ErrOut, name)
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(f.IOStreams, result.Items)
	}

	tp := output.NewTablePrinter(f.IOStreams)
	tp.AddHeader("NAME", "STATE", "API_ID", "DEFAULT")
	for _, dto := range result.Items {
		def := ""
		if dto.Default {
			def = "*"
		}
		apiID := ""
		if dto.APIID != nil {
			apiID = strconv.Itoa(*dto.APIID)
		}
		tp.AddRow(dto.Name, dto.State, apiID, def)
	}
	return tp.Render()
}
