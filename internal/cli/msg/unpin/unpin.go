// Package unpin implements "tg msg unpin" as a thin wrapper around pin.
package unpin

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/cli/msg/pin"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// New builds the cobra command for "tg msg unpin".
func New(f *runtime.Invocation, runF func(*pin.Options) error) *cobra.Command {
	return pin.NewUnpin(f, runF)
}
