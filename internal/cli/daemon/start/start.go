// Package start implements "tg daemon start".
package start

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

// New builds the cobra command for "tg daemon start".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Ask the host service manager to start the per-account daemon",
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

// Run starts the service for opts.Account.
func Run(_ context.Context, opts *Options) error {
	if opts.Account == "" {
		return fmt.Errorf("%w: start requires an account", command.ErrUsage)
	}
	mgr, err := opts.NewMgr(opts.Account)
	if err != nil {
		return err
	}
	st, _ := mgr.Status()
	if st == nil || !st.Installed {
		return fmt.Errorf("%w: daemon is not installed for %q (run: tg daemon install)",
			command.ErrPrecondition, opts.Account)
	}
	if err := mgr.Start(); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(opts.IOStreams.ErrOut, "tg daemon started for account %q\n", opts.Account)
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
