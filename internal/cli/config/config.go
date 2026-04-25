package config

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/cli/config/edit"
	"github.com/vika2603/telegram-cli/internal/cli/config/get"
	"github.com/vika2603/telegram-cli/internal/cli/config/path"
	"github.com/vika2603/telegram-cli/internal/cli/config/set"
	"github.com/vika2603/telegram-cli/internal/cli/config/show"
	"github.com/vika2603/telegram-cli/internal/cli/config/unset"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// New constructs the "tg config" group and wires its subcommands.
func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Short:   "Inspect configuration",
		GroupID: "setup",
	}
	cmd.AddCommand(edit.New(f, nil))
	cmd.AddCommand(get.New(f, nil))
	cmd.AddCommand(path.New(f, nil))
	cmd.AddCommand(set.New(f, nil))
	cmd.AddCommand(show.New(f, nil))
	cmd.AddCommand(unset.New(f, nil))
	return cmd
}
