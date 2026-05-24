// Package me implements the "tg me" single command: print my own
// Telegram identity (read-only; no positional args).
package me

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionme "github.com/vika2603/telegram-cli/internal/action/me"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

type Options struct {
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Fetch     func(ctx context.Context) (output.UserRow, error)
}

func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}

	cmd := &cobra.Command{
		Use:     "me",
		Short:   "Print my Telegram identity",
		GroupID: "core",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Fetch = newFetch(f)
			return Run(cmd.Context(), opts)
		},
	}

	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"id", "username", "first_name", "last_name", "phone", "is_bot", "is_self", "is_verified"})
	return cmd
}

func Run(ctx context.Context, opts *Options) error {
	if opts.Fetch == nil {
		return fmt.Errorf("%w: internal error: me fetch function is not configured", command.ErrPrecondition)
	}
	row, err := actionme.Show(ctx, opts.Fetch)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderUser(opts.IOStreams, row)
}

func newFetch(f *runtime.Invocation) func(context.Context) (output.UserRow, error) {
	return func(ctx context.Context) (output.UserRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.UserRow{}, err
		}
		// Daemon fast-path: if a per-account socket is reachable and
		// the user has not opted out, run me.show over IPC and skip
		// the dial/auth/resume round trip entirely.
		if cl, _ := runtime.MaybeDialDaemon(ctx, f, acct); cl != nil {
			defer func() { _ = cl.Close() }()
			raw, err := cl.Call(ctx, "me.show", nil)
			if err != nil {
				return output.UserRow{}, err
			}
			var row output.UserRow
			if err := json.Unmarshal(raw, &row); err != nil {
				return output.UserRow{}, err
			}
			return row, nil
		}

		var row output.UserRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
				var err error
				row, err = telegram.ShowMe(ctx, api)
				return err
			})
		return row, err
	}
}
