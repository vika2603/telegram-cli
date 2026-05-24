// Package install implements "tg daemon install".
package install

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
	Account    string
	BinaryPath string
	LogFile    string
	LogMaxMB   int
	Force      bool

	IOStreams *ui.IOStreams
	NewMgr    func(account string) (daemon.Manager, error)
}

// New builds the cobra command for "tg daemon install".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Register the per-account daemon with the host service manager",
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
	cmd.Flags().StringVar(&opts.BinaryPath, "binary", "", "Path to the tg binary to register (default: current executable)")
	cmd.Flags().StringVar(&opts.LogFile, "log-file", "", "Daemon log path (default: <account-dir>/daemon/daemon.log)")
	cmd.Flags().IntVar(&opts.LogMaxMB, "log-max-mb", 10, "Daemon log rotation threshold in MB")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Reinstall over an existing registration")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, SkipAuthCheck: true})
	return cmd
}

// Run installs the service definition for opts.Account.
func Run(ctx context.Context, opts *Options) error {
	if opts.Account == "" {
		return fmt.Errorf("%w: install requires an account (use --account or default slot)", command.ErrUsage)
	}
	mgr, err := opts.NewMgr(opts.Account)
	if err != nil {
		return err
	}

	st, _ := mgr.Status()
	if st != nil && st.Installed && !opts.Force {
		return fmt.Errorf("%w: daemon already installed for %q; pass --force to reinstall",
			command.ErrPrecondition, opts.Account)
	}

	cfg := daemon.Config{
		Account:    opts.Account,
		BinaryPath: opts.BinaryPath,
		LogFile:    opts.LogFile,
		LogMaxSize: int64(opts.LogMaxMB) * 1024 * 1024,
	}
	if err := daemon.Resolve(&cfg); err != nil {
		return err
	}
	if err := mgr.Install(cfg); err != nil {
		return fmt.Errorf("install: %w", err)
	}

	_ = daemon.SaveMeta(&daemon.Meta{
		Account:     opts.Account,
		LogFile:     cfg.LogFile,
		LogMaxSize:  cfg.LogMaxSize,
		BinaryPath:  cfg.BinaryPath,
		InstalledAt: daemon.NowISO(),
		Platform:    mgr.Platform(),
		SocketPath:  daemon.SocketPath(opts.Account),
	})

	io := opts.IOStreams
	_, _ = fmt.Fprintln(io.ErrOut, "tg daemon installed.")
	_, _ = fmt.Fprintf(io.ErrOut, "  Account:   %s\n", opts.Account)
	_, _ = fmt.Fprintf(io.ErrOut, "  Platform:  %s\n", mgr.Platform())
	_, _ = fmt.Fprintf(io.ErrOut, "  Binary:    %s\n", cfg.BinaryPath)
	_, _ = fmt.Fprintf(io.ErrOut, "  Log:       %s\n", cfg.LogFile)
	_, _ = fmt.Fprintf(io.ErrOut, "  Updates:   %s\n", daemon.UpdatesFile(opts.Account))
	if enabled, user := daemon.CheckLinger(); !enabled && user != "" {
		_, _ = fmt.Fprintln(io.ErrOut)
		_, _ = fmt.Fprintln(io.ErrOut, "Warning: systemd user linger is not enabled.")
		_, _ = fmt.Fprintf(io.ErrOut, "  The daemon will stop when %s logs out.\n", user)
		_, _ = fmt.Fprintf(io.ErrOut, "  Run: sudo loginctl enable-linger %s\n", user)
	}
	return nil
}

// effectiveAccount picks the account the user wants — explicit
// --account if set, else default slot. Returning "" lets Run surface a
// usage error with the right code.
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
