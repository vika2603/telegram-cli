// Package set implements "tg password set".
package set

import (
	"context"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionpassword "github.com/vika2603/telegram-cli/internal/action/password"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	Hint          string
	RecoveryEmail string
	CurrentStdin  bool
	NewStdin      bool

	F        *runtime.Invocation
	Exporter output.Exporter

	// Check reports whether the account currently has a password set.
	Check actionpassword.CheckFunc

	// Apply performs the actual SRP ceremony + password update.
	Apply actionpassword.SetFunc
}

// New builds the cobra command for "tg password set".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{F: f}
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set or change the 2FA password",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if runF != nil {
				return runF(opts)
			}
			opts.Check = newCheck(f)
			opts.Apply = newApply(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.Hint, "hint", "", "New password hint (visible on the Telegram lock screen)")
	cmd.Flags().StringVar(&opts.RecoveryEmail, "recovery-email", "", "Recovery email (currently informational)")
	cmd.Flags().BoolVar(&opts.CurrentStdin, "current-stdin", false, "Read current password from stdin (first line)")
	cmd.Flags().BoolVar(&opts.NewStdin, "new-stdin", false, "Read new password from stdin (next line)")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "had_previous", "has_hint", "has_recovery_email"})
	return cmd
}

// Run validates options and performs the password set operation.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionpassword.Set(ctx, actionpassword.SetRequest{
		Hint:          opts.Hint,
		RecoveryEmail: opts.RecoveryEmail,
		CurrentStdin:  opts.CurrentStdin,
		NewStdin:      opts.NewStdin,
		IOStreams:     opts.F.IOStreams,
		Prompter:      opts.F.Prompter,
	}, opts.Check, opts.Apply)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.F.IOStreams, row)
	}
	return output.WriteAccountPasswordJSON(opts.F.IOStreams.Out, row)
}

func newCheck(f *runtime.Invocation) actionpassword.CheckFunc {
	return func(ctx context.Context) (bool, error) {
		acct, err := f.Account("")
		if err != nil {
			return false, err
		}
		var enabled bool
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
				var err error
				enabled, err = telegram.PasswordEnabled(ctx, api)
				return err
			})
		return enabled, err
	}
}

func newApply(f *runtime.Invocation) actionpassword.SetFunc {
	return func(ctx context.Context, cur, next, hint string) (bool, error) {
		acct, err := f.Account("")
		if err != nil {
			return false, err
		}
		var hadPrevious bool
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
				var err error
				hadPrevious, err = telegram.SetPassword(ctx, api, cur, next, hint)
				return err
			})
		return hadPrevious, err
	}
}
