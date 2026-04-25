// Package path implements the "tg config path" command.
package path

import (
	"fmt"

	"github.com/spf13/cobra"

	actionconfig "github.com/vika2603/telegram-cli/internal/action/config"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// Options holds flag values and injectable dependencies for "config path".
type Options struct {
	Exporter output.Exporter
}

// New constructs the cobra.Command for "config path".
// When runF is nil, production logic (pathRun) is used.
// Tests pass a capture lambda to verify flag parsing without touching disk.
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}

	cmd := &cobra.Command{
		Use:          "path",
		Short:        "Print the resolved config file path",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return pathRun(f, cmd, opts)
		},
	}

	command.SetMeta(cmd, command.Meta{SkipAuthCheck: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"path"})

	return cmd
}

func pathRun(f *runtime.Invocation, cmd *cobra.Command, opts *Options) error {
	flagConfigPath, _ := cmd.Flags().GetString("config")
	resolvedPath := actionconfig.Path(cmd.Context(), actionconfig.PathRequest{FlagPath: flagConfigPath})

	if opts.Exporter != nil {
		return opts.Exporter.Write(f.IOStreams, actionconfig.PathResult{Path: resolvedPath})
	}

	quiet, _ := cmd.Flags().GetBool("quiet")
	if quiet {
		return nil
	}

	_, _ = fmt.Fprintln(f.IOStreams.Out, resolvedPath)
	return nil
}
