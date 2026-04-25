// Package completion implements `tg completion [shell]`, producing a
// script you can `source` for tab completion.
package completion

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion scripts",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		GroupID:   "setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(f.IOStreams.Out)
			case "zsh":
				return root.GenZshCompletion(f.IOStreams.Out)
			case "fish":
				return root.GenFishCompletion(f.IOStreams.Out, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(f.IOStreams.Out)
			}
			return nil
		},
	}
	command.SetMeta(cmd, command.Meta{SkipAuthCheck: true})
	return cmd
}
