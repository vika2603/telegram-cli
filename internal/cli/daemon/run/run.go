// Package run implements the hidden "tg daemon run" worker entry. The
// OS service definition installed by "tg daemon install" invokes this
// subcommand; humans should not run it directly.
package run

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/daemon"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options is plumbed for test injection. The hidden command itself has
// no flags.
type Options struct {
	IOStreams *ui.IOStreams
	Inv       *runtime.Invocation
}

// New builds the hidden cobra command for "tg daemon run".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{Inv: f}
	cmd := &cobra.Command{
		Use:    "run",
		Short:  "Worker entry invoked by the OS service definition",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			return Run(cmd.Context(), opts)
		},
	}
	// NeedsAccount + NeedsClient so the PersistentPreRunE wires up the
	// account and the auth check runs before we try to dial; the worker
	// loop itself opens the client via Inv.WithClient.
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	return cmd
}

// Run dispatches to daemon.Run for the resolved account.
func Run(ctx context.Context, opts *Options) error {
	if opts.Inv == nil || opts.Inv.Account == nil {
		return fmt.Errorf("%w: daemon run requires runtime account accessor",
			command.ErrPrecondition)
	}
	acct, err := opts.Inv.Account("")
	if err != nil {
		return err
	}
	return daemon.Run(ctx, daemon.WorkerOptions{
		Inv:       opts.Inv,
		Account:   acct,
		IOStreams: opts.IOStreams,
	})
}
