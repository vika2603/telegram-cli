// Package list implements "tg session list".
package list

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionsession "github.com/vika2603/telegram-cli/internal/action/session"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	F        *runtime.Invocation
	Exporter output.Exporter

	// Fetch is the closure that performs the actual Telegram call.
	// Production code sets it via newFetch; tests stub it directly.
	Fetch actionsession.FetchFunc
}

// New builds the cobra command for "tg session list".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{F: f}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List active remote sessions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if runF != nil {
				return runF(opts)
			}
			opts.Fetch = newFetch()
			return Run(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{
		"hash", "current", "device_model", "platform", "app_name", "app_version",
		"country", "ip", "date_created", "date_active",
	})
	return cmd
}

// Run validates options and emits one ndjson row per authorization.
func Run(ctx context.Context, opts *Options) error {
	if opts.Fetch == nil {
		return fmt.Errorf("%w: internal error: session list fetch function is not configured", command.ErrPrecondition)
	}
	acct, err := opts.F.Account("")
	if err != nil {
		return err
	}
	var rows []output.AccountSessionRow
	err = opts.F.WithPeers(ctx, acct, runtime.ClientOptsFrom(opts.F, acct),
		func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
			var ferr error
			rows, ferr = actionsession.List(ctx, api, opts.Fetch)
			return ferr
		})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if opts.Exporter != nil {
			if werr := opts.Exporter.Write(opts.F.IOStreams, row); werr != nil {
				return werr
			}
			continue
		}
		if werr := output.WriteAccountSessionJSON(opts.F.IOStreams.Out, row); werr != nil {
			return werr
		}
	}
	return nil
}

// newFetch is the production closure. Tests stub it directly at the CLI edge.
func newFetch() actionsession.FetchFunc {
	return func(ctx context.Context, api *tg.Client) (*tg.AccountAuthorizations, error) {
		return api.AccountGetAuthorizations(ctx)
	}
}
