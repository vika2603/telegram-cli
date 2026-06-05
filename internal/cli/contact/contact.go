// Package contact hosts the "tg contact" command group.
package contact

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/cli/contact/add"
	"github.com/vika2603/telegram-cli/internal/cli/contact/block"
	del "github.com/vika2603/telegram-cli/internal/cli/contact/delete"
	"github.com/vika2603/telegram-cli/internal/cli/contact/list"
	"github.com/vika2603/telegram-cli/internal/cli/contact/report"
	"github.com/vika2603/telegram-cli/internal/cli/contact/unblock"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// New builds the cobra command group for "tg contact".
func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "contact",
		Short:   "Address book: list, add, delete, block, unblock, report",
		GroupID: "core",
	}
	cmd.AddCommand(list.New(f, nil))
	cmd.AddCommand(add.New(f, nil))
	cmd.AddCommand(del.New(f, nil))
	cmd.AddCommand(block.New(f, nil))
	cmd.AddCommand(unblock.New(f, nil))
	cmd.AddCommand(report.New(f, nil))
	return cmd
}
