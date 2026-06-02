// Package msg hosts the "tg msg" command group.
package msg

import (
	"github.com/spf13/cobra"

	del "github.com/vika2603/telegram-cli/internal/cli/msg/delete"
	"github.com/vika2603/telegram-cli/internal/cli/msg/download"
	"github.com/vika2603/telegram-cli/internal/cli/msg/edit"
	"github.com/vika2603/telegram-cli/internal/cli/msg/forward"
	"github.com/vika2603/telegram-cli/internal/cli/msg/link"
	"github.com/vika2603/telegram-cli/internal/cli/msg/list"
	"github.com/vika2603/telegram-cli/internal/cli/msg/pin"
	"github.com/vika2603/telegram-cli/internal/cli/msg/react"
	"github.com/vika2603/telegram-cli/internal/cli/msg/schedulecancel"
	"github.com/vika2603/telegram-cli/internal/cli/msg/schedulelist"
	"github.com/vika2603/telegram-cli/internal/cli/msg/send"
	"github.com/vika2603/telegram-cli/internal/cli/msg/sticker"
	"github.com/vika2603/telegram-cli/internal/cli/msg/unpin"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// New builds the cobra command group for "tg msg".
func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "msg",
		Short:   "Messages: send, media, edit, delete, forward, react, pin/unpin, schedule, list, link",
		GroupID: "core",
	}
	cmd.AddCommand(list.New(f, nil))
	cmd.AddCommand(link.New(f, nil))
	cmd.AddCommand(download.New(f, nil))
	cmd.AddCommand(send.New(f, nil))
	cmd.AddCommand(sticker.New(f))
	cmd.AddCommand(edit.New(f, nil))
	cmd.AddCommand(del.New(f, nil))
	cmd.AddCommand(forward.New(f, nil))
	cmd.AddCommand(react.New(f, nil))
	cmd.AddCommand(pin.New(f, nil))
	cmd.AddCommand(unpin.New(f, nil))
	cmd.AddCommand(schedulelist.New(f, nil))
	cmd.AddCommand(schedulecancel.New(f, nil))
	return cmd
}
