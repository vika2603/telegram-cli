// Package disable implements "tg password disable".
package disable

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
	CurrentStdin bool
	Yes          bool

	F        *runtime.Invocation
	Exporter output.Exporter

	// Check reports whether the account currently has a password set.
	Check actionpassword.CheckFunc

	// Apply issues the disable request; wraps bad-password errors with
	// ErrBadPassword.
	Apply actionpassword.DisableFunc
}

// New builds the cobra command for "tg password disable".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{F: f}
	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Remove the 2FA password from the account",
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
	cmd.Flags().BoolVar(&opts.CurrentStdin, "current-stdin", false, "Read current password from stdin (first line)")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip destructive confirmation prompt")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action"})
	return cmd
}

// Run validates options and performs the password disable operation.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionpassword.Disable(ctx, actionpassword.DisableRequest{
		CurrentStdin: opts.CurrentStdin,
		Yes:          opts.Yes,
		IOStreams:    opts.F.IOStreams,
		Prompter:     opts.F.Prompter,
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

func newApply(f *runtime.Invocation) actionpassword.DisableFunc {
	return func(ctx context.Context, cur string) error {
		acct, err := f.Account("")
		if err != nil {
			return err
		}
		return f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
				return telegram.DisablePassword(ctx, api, cur)
			})
	}
}
