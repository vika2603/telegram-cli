// Package add implements "tg contact add <phone>".
package add

import (
	"context"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actioncontact "github.com/vika2603/telegram-cli/internal/action/contact"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	Phone  string
	First  string
	Last   string
	Mutual bool

	IOStreams *ui.IOStreams
	Exporter  output.Exporter

	// Add is the closure that performs the actual Telegram call. Production
	// code sets it via newAdd; tests stub it directly.
	Add actioncontact.AddFunc
}

// New builds the cobra command for "tg contact add".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "add <phone>",
		Short: "Add a contact by phone number",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Phone = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Add = newAdd(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.First, "first", "", "First name (required)")
	cmd.Flags().StringVar(&opts.Last, "last", "", "Last name")
	cmd.Flags().BoolVar(&opts.Mutual, "mutual", false, "Share my phone number")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{
		"id", "first_name", "last_name", "username", "phone", "mutual", "bot",
	})
	return cmd
}

// Run collects the raw request, delegates validation/mutation, and renders rows.
func Run(ctx context.Context, opts *Options) error {
	row, err := actioncontact.Add(ctx, actioncontact.AddRequest{
		Phone:  opts.Phone,
		First:  opts.First,
		Last:   opts.Last,
		Mutual: opts.Mutual,
	}, opts.Add)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderContacts(opts.IOStreams, []output.ContactRow{row})
}

// newAdd returns the production Add closure that calls the Telegram API.
func newAdd(f *runtime.Invocation) actioncontact.AddFunc {
	return func(ctx context.Context, q actioncontact.AddQuery) (output.ContactRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.ContactRow{}, err
		}
		var row output.ContactRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, pm *peers.Manager, _ *peer.Resolver) error {
				r, ferr := telegram.AddContact(ctx, api, pm, q)
				row = r
				return ferr
			})
		return row, err
	}
}
