// Package topic implements the "tg chat topic" command group.
package topic

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/runtime"
)

// New builds the "tg chat topic" command group.
func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "topic",
		Short: "Forum topics of a supergroup",
	}
	cmd.AddCommand(newList(f, nil))
	cmd.AddCommand(newCreate(f, nil))
	cmd.AddCommand(newEdit(f, nil))
	cmd.AddCommand(newDeleteTopic(f, nil))
	cmd.AddCommand(newPinTopic(f, nil))
	cmd.AddCommand(newUnpinTopic(f, nil))
	cmd.AddCommand(newTopicInfo(f, nil))
	cmd.AddCommand(newMuteTopic(f, nil))
	cmd.AddCommand(newUnmuteTopic(f, nil))
	cmd.AddCommand(newReadTopic(f, nil))
	return cmd
}
