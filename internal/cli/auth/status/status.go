// Package status implements "tg auth status [name] [--probe]".
package status

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/account"
	actionauth "github.com/vika2603/telegram-cli/internal/action/auth"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	Name  string
	Probe bool

	F        *runtime.Invocation
	Exporter output.Exporter

	// DoProbe runs a liveness check against the Telegram API.
	// Nil means use the production closure (newDoProbe).
	// Tests stub directly.
	DoProbe func(ctx context.Context, api *tg.Client) error
}

// New builds the cobra command for "tg auth status [name]".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{F: f}
	cmd := &cobra.Command{
		Use:   "status [name]",
		Short: "Show local health of the resolved account slot",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				opts.Name = args[0]
			}
			if runF != nil {
				return runF(opts)
			}
			opts.DoProbe = newDoProbe()
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.Probe, "probe", false,
		"Run a live UsersGetFullUser check to test session liveness")
	// AccountFromArg: positional [name] selects the slot. Root pre-runE must
	// NOT pre-load the default slot — the command body resolves via
	// f.Account(opts.Name).
	command.SetMeta(cmd, command.Meta{AccountFromArg: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{
		"name", "state", "api_id", "default", "probed", "probe_ok", "session_modified",
	})
	return cmd
}

// Run executes the status operation.
func Run(ctx context.Context, opts *Options) error {
	result, err := actionauth.Status(ctx, actionauth.StatusRequest{
		Name:  opts.Name,
		Probe: opts.Probe,
	}, actionauth.StatusDeps{
		ResolveAccount:  opts.F.Account,
		Config:          opts.F.Config,
		SessionModified: actionauth.DefaultSessionModified,
		Probe: func(ctx context.Context, acct *account.Account) error {
			return opts.F.WithClient(ctx, acct, runtime.ClientOptsFrom(opts.F, acct),
				func(ctx context.Context, cl session.Client) error {
					if opts.DoProbe == nil {
						opts.DoProbe = newDoProbe()
					}
					return opts.DoProbe(ctx, tg.NewClient(cl.Invoker()))
				},
			)
		},
	})
	if err != nil {
		return err
	}
	row := result.Row
	if result.ProbeSkipped {
		_, _ = fmt.Fprintf(opts.F.IOStreams.ErrOut,
			"skipping probe (state=%s)\n", result.ProbeSkippedState)
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.F.IOStreams, row)
	}

	// Human fallback: one key per line.
	out := opts.F.IOStreams.Out
	_, _ = fmt.Fprintf(out, "name: %s\n", row.Name)
	_, _ = fmt.Fprintf(out, "state: %s\n", row.State)
	_, _ = fmt.Fprintf(out, "api_id: %d\n", row.APIID)
	_, _ = fmt.Fprintf(out, "default: %t\n", row.Default)
	sm := row.SessionModified
	if sm == "" {
		sm = "(none)"
	}
	_, _ = fmt.Fprintf(out, "session_modified: %s\n", sm)
	if opts.Probe {
		_, _ = fmt.Fprintf(out, "probed: %t\n", row.Probed)
		_, _ = fmt.Fprintf(out, "probe_ok: %t\n", row.ProbeOK)
	}
	return nil
}

// newDoProbe returns the production DoProbe closure.
func newDoProbe() func(ctx context.Context, api *tg.Client) error {
	return func(ctx context.Context, api *tg.Client) error {
		_, err := api.UsersGetFullUser(ctx, &tg.InputUserSelf{})
		return err
	}
}
