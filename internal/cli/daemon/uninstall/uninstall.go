// Package uninstall implements "tg daemon uninstall".
package uninstall

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/daemon"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	Account   string
	IOStreams *ui.IOStreams
	NewMgr    func(account string) (daemon.Manager, error)
}

// New builds the cobra command for "tg daemon uninstall".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the per-account daemon registration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IOStreams = f.IOStreams
			opts.Account = effectiveAccount(f)
			if runF != nil {
				return runF(opts)
			}
			opts.NewMgr = daemon.NewManager
			return Run(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, SkipAuthCheck: true})
	return cmd
}

// Run removes the service definition for opts.Account. Uninstall is
// idempotent — a missing service is not an error, only logged.
func Run(_ context.Context, opts *Options) error {
	if opts.Account == "" {
		return fmt.Errorf("%w: uninstall requires an account", command.ErrUsage)
	}
	mgr, err := opts.NewMgr(opts.Account)
	if err != nil {
		return err
	}
	if err := mgr.Uninstall(); err != nil {
		return fmt.Errorf("uninstall: %w", err)
	}
	_ = daemon.RemoveMeta(opts.Account)
	_, _ = fmt.Fprintf(opts.IOStreams.ErrOut, "tg daemon removed for account %q\n", opts.Account)
	return nil
}

func effectiveAccount(f *runtime.Invocation) string {
	if f.AccountName != "" {
		return f.AccountName
	}
	if f.Account == nil {
		return ""
	}
	acct, err := f.Account("")
	if err != nil || acct == nil {
		return ""
	}
	return acct.Meta.Name
}
