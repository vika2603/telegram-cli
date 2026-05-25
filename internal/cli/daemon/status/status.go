// Package status implements "tg daemon status".
package status

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/daemon"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	Account   string
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	NewMgr    func(account string) (daemon.Manager, error)
}

// New builds the cobra command for "tg daemon status".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the per-account daemon's installation and run state",
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
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"account", "installed", "running", "pid", "platform", "log_file", "updates_file", "socket_path", "installed_at", "stats"})
	return cmd
}

// statusRow is the JSON shape emitted by "tg daemon status --json".
// Mirrors daemon.Status fields plus the on-disk meta sidecar, with
// live metrics merged in when the per-account socket is reachable.
type statusRow struct {
	Account     string                  `json:"account"`
	Installed   bool                    `json:"installed"`
	Running     bool                    `json:"running"`
	PID         int                     `json:"pid,omitempty"`
	Platform    string                  `json:"platform"`
	LogFile     string                  `json:"log_file,omitempty"`
	UpdatesFile string                  `json:"updates_file,omitempty"`
	SocketPath  string                  `json:"socket_path,omitempty"`
	InstalledAt string                  `json:"installed_at,omitempty"`
	Stats       *daemon.MetricsSnapshot `json:"stats,omitempty"`
}

// Run prints status to stderr (human) or stdout (json).
func Run(ctx context.Context, opts *Options) error {
	if opts.Account == "" {
		return fmt.Errorf("%w: status requires an account", command.ErrUsage)
	}
	mgr, err := opts.NewMgr(opts.Account)
	if err != nil {
		return err
	}
	st, err := mgr.Status()
	if err != nil {
		return err
	}
	if st == nil {
		st = &daemon.Status{Account: opts.Account, Platform: mgr.Platform()}
	}

	row := statusRow{
		Account:     st.Account,
		Installed:   st.Installed,
		Running:     st.Running,
		PID:         st.PID,
		Platform:    st.Platform,
		UpdatesFile: daemon.UpdatesFile(opts.Account),
		SocketPath:  daemon.SocketPath(opts.Account),
	}
	if meta, err := daemon.LoadMeta(opts.Account); err == nil && meta != nil {
		row.LogFile = meta.LogFile
		row.InstalledAt = meta.InstalledAt
	}

	// Pull live metrics over the socket when the daemon is up. Failure
	// here is non-fatal — the OS-level status still renders. Common
	// causes: daemon is bootstrapped but the socket is not yet open
	// (race against worker startup) or schema mismatch.
	if daemon.DaemonReachable(opts.Account) {
		if cl, err := daemon.Dial(ctx, opts.Account); err == nil {
			if snap, err := cl.Stats(ctx); err == nil {
				row.Stats = &snap
			}
			_ = cl.Close()
		}
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return renderHuman(opts.IOStreams, row)
}

func renderHuman(io *ui.IOStreams, r statusRow) error {
	enc := json.NewEncoder(io.Out)
	enc.SetIndent("", "  ")
	// Even in human mode we dump JSON since this command's primary
	// audience is scripts. The reasoning matches inbox/digest: a small,
	// structured status is more useful than a hand-formatted table.
	return enc.Encode(r)
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
