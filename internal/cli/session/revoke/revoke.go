// Package revoke implements "tg session revoke".
package revoke

import (
	"context"
	"encoding/json"
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
	Hash      string
	AllOthers bool
	Yes       bool

	F        *runtime.Invocation
	Exporter output.Exporter

	// Fetch pulls the current list of authorizations. Used for the safety
	// check AND the rich prompt content.
	Fetch actionsession.FetchFunc
	// Reset terminates a single session by hash.
	Reset actionsession.ResetFunc
	// ResetAll terminates every session in the given hash list.
	// Production code calls AccountResetAuthorization for each victim hash.
	ResetAll actionsession.ResetAllFunc
}

// New builds the cobra command for "tg session revoke".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{F: f}
	cmd := &cobra.Command{
		Use:   "revoke [hash]",
		Short: "Revoke remote sessions",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				opts.Hash = args[0]
			}
			if runF != nil {
				return runF(opts)
			}
			opts.Fetch = defaultFetch
			opts.Reset = defaultReset
			opts.ResetAll = defaultResetAll
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.AllOthers, "all-others", false, "Terminate every session except the current one")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "hash", "device", "count", "all_others", "kept_hash"})
	return cmd
}

// Run opens the current account client, dispatches the action, and emits the result.
func Run(ctx context.Context, opts *Options) error {
	if opts.Fetch == nil || opts.Reset == nil || opts.ResetAll == nil {
		return fmt.Errorf("%w: internal error: session revoke functions are not configured", command.ErrPrecondition)
	}
	req := actionsession.TerminateRequest{
		Hash:      opts.Hash,
		AllOthers: opts.AllOthers,
		Yes:       opts.Yes,
		Prompter:  opts.F.Prompter,
	}
	if err := actionsession.ValidateTerminate(req); err != nil {
		return err
	}

	acct, err := opts.F.Account("")
	if err != nil {
		return err
	}
	var row map[string]any
	err = opts.F.WithPeers(ctx, acct, runtime.ClientOptsFrom(opts.F, acct),
		func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
			var ferr error
			row, ferr = actionsession.Terminate(ctx, api, req, opts.Fetch, opts.Reset, opts.ResetAll)
			return ferr
		})
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.F.IOStreams, row)
	}
	b, err := json.Marshal(row)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(opts.F.IOStreams.Out, "%s\n", b)
	return nil
}

// defaultFetch/defaultReset/defaultResetAll are the production closures.
func defaultFetch(ctx context.Context, api *tg.Client) (*tg.AccountAuthorizations, error) {
	return api.AccountGetAuthorizations(ctx)
}

func defaultReset(ctx context.Context, api *tg.Client, hash int64) error {
	_, err := api.AccountResetAuthorization(ctx, hash)
	return err
}

// defaultResetAll terminates each victim session individually.
// The Telegram API has no bulk-revoke-all-others endpoint in gotd/td v0.107.0;
// AccountResetAuthorization is called once per victim hash.
func defaultResetAll(ctx context.Context, api *tg.Client, victimHashes []int64) error {
	for _, h := range victimHashes {
		if _, err := api.AccountResetAuthorization(ctx, h); err != nil {
			return err
		}
	}
	return nil
}
