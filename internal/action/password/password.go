// Package password contains 2FA password command actions.
package password

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// CheckFunc reports whether the account currently has a 2FA password.
type CheckFunc func(context.Context) (bool, error)

// SetFunc applies a new password. It returns whether the account had a
// previous password according to Telegram's latest SRP state.
type SetFunc func(context.Context, string, string, string) (bool, error)

// DisableFunc removes the existing password.
type DisableFunc func(context.Context, string) error

// SetRequest is the raw request for `tg password set`.
type SetRequest struct {
	Hint          string
	RecoveryEmail string
	CurrentStdin  bool
	NewStdin      bool
	IOStreams     *ui.IOStreams
	Prompter      ui.Prompter
}

// DisableRequest is the raw request for `tg password disable`.
type DisableRequest struct {
	CurrentStdin bool
	Yes          bool
	IOStreams    *ui.IOStreams
	Prompter     ui.Prompter
}

// Set validates inputs, reads passwords, and applies the new 2FA password.
func Set(ctx context.Context, req SetRequest, check CheckFunc, apply SetFunc) (output.AccountPasswordRow, error) {
	if check == nil {
		return output.AccountPasswordRow{}, fmt.Errorf("%w: password set called without check function", command.ErrPrecondition)
	}
	if apply == nil {
		return output.AccountPasswordRow{}, fmt.Errorf("%w: password set called without apply function", command.ErrPrecondition)
	}
	hasPwd, err := check(ctx)
	if err != nil {
		return output.AccountPasswordRow{}, err
	}
	var cur string
	if hasPwd {
		c, err := readCurrentPassword(req.IOStreams, req.Prompter, req.CurrentStdin)
		if err != nil {
			return output.AccountPasswordRow{}, err
		}
		cur = c
	} else if req.CurrentStdin {
		return output.AccountPasswordRow{}, fmt.Errorf("%w: --current-stdin given but account has no current password", command.ErrUsage)
	}
	next, err := readNewPassword(req.IOStreams, req.Prompter, req.NewStdin)
	if err != nil {
		return output.AccountPasswordRow{}, err
	}
	hadPrevious, err := apply(ctx, cur, next, req.Hint)
	if err != nil {
		return output.AccountPasswordRow{}, err
	}
	return output.AccountPasswordRow{
		Action:           "password_set",
		HadPrevious:      hadPrevious,
		HasHint:          req.Hint != "",
		HasRecoveryEmail: req.RecoveryEmail != "",
	}, nil
}

// Disable validates inputs, confirms, and removes the 2FA password.
func Disable(ctx context.Context, req DisableRequest, check CheckFunc, apply DisableFunc) (output.AccountPasswordRow, error) {
	if check == nil {
		return output.AccountPasswordRow{}, fmt.Errorf("%w: password disable called without check function", command.ErrPrecondition)
	}
	if apply == nil {
		return output.AccountPasswordRow{}, fmt.Errorf("%w: password disable called without apply function", command.ErrPrecondition)
	}
	hasPwd, err := check(ctx)
	if err != nil {
		return output.AccountPasswordRow{}, err
	}
	if !hasPwd {
		return output.AccountPasswordRow{}, fmt.Errorf("%w: no password set; nothing to disable", command.ErrPrecondition)
	}
	cur, err := readCurrentPassword(req.IOStreams, req.Prompter, req.CurrentStdin)
	if err != nil {
		return output.AccountPasswordRow{}, err
	}
	if err := ui.ConfirmDestructive(req.Prompter, "Disable 2FA password?", req.Yes); err != nil {
		return output.AccountPasswordRow{}, err
	}
	if err := apply(ctx, cur); err != nil {
		return output.AccountPasswordRow{}, err
	}
	return output.AccountPasswordRow{Action: "password_disable"}, nil
}

func readCurrentPassword(streams *ui.IOStreams, prompter ui.Prompter, currentStdin bool) (string, error) {
	if currentStdin {
		return readLine(streams.In)
	}
	if prompter == nil {
		return "", fmt.Errorf("%w: no prompter available; pass --current-stdin", command.ErrPrecondition)
	}
	return prompter.Password("Current 2FA password")
}

func readNewPassword(streams *ui.IOStreams, prompter ui.Prompter, newStdin bool) (string, error) {
	if newStdin {
		return readLine(streams.In)
	}
	if prompter == nil {
		return "", fmt.Errorf("%w: no prompter available; pass --new-stdin", command.ErrPrecondition)
	}
	next, err := prompter.Password("New 2FA password")
	if err != nil {
		return "", err
	}
	confirm, err := prompter.Password("Confirm new password")
	if err != nil {
		return "", err
	}
	if next != confirm {
		return "", fmt.Errorf("%w: passwords do not match", command.ErrUsage)
	}
	return next, nil
}

func readLine(r io.Reader) (string, error) {
	var buf []byte
	b := make([]byte, 1)
	for {
		n, err := r.Read(b)
		if n > 0 {
			if b[0] == '\n' {
				break
			}
			buf = append(buf, b[0])
		}
		if err != nil {
			if len(buf) == 0 {
				return "", err
			}
			break
		}
	}
	return strings.TrimRight(string(buf), "\r"), nil
}
