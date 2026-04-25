package ui

import (
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
)

// ConfirmDestructive asks the user to confirm a destructive action unless
// `yes` is true. Returns ErrNotConfirmed on "no", or wraps the prompter's
// error on any transport failure.
func ConfirmDestructive(prompter Prompter, message string, yes bool) error {
	if yes {
		return nil
	}
	if prompter == nil {
		return fmt.Errorf("%w: no prompter available for confirmation", command.ErrPrecondition)
	}
	ok, err := prompter.Confirm(message, false)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w", command.ErrNotConfirmed)
	}
	return nil
}
