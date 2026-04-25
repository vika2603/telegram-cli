// Package list implements "tg contact list".
package list

import (
	"context"
	"strconv"
	"strings"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/account"
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
	Blocked    bool
	MutualOnly bool
	Bots       bool

	IOStreams *ui.IOStreams
	Exporter  output.Exporter

	// Fetch is the closure that performs the actual Telegram call. Production
	// code sets it via newFetch; tests stub it directly.
	Fetch actioncontact.ListFunc
}

// New builds the cobra command for "tg contact list".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List my contacts (or blocked users with --blocked)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Fetch = newFetch(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.Blocked, "blocked", false, "List blocked users instead")
	cmd.Flags().BoolVar(&opts.MutualOnly, "mutual-only", false, "Filter to mutual contacts")
	cmd.Flags().BoolVar(&opts.Bots, "bots", false, "Include bots")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{
		"id", "first_name", "last_name", "username", "phone", "mutual", "blocked", "bot",
	})
	return cmd
}

// Run collects the raw request, delegates validation/loading, and renders rows.
func Run(ctx context.Context, opts *Options) error {
	rows, err := actioncontact.List(ctx, actioncontact.ListRequest{
		Blocked:    opts.Blocked,
		MutualOnly: opts.MutualOnly,
		Bots:       opts.Bots,
	}, opts.Fetch)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	return output.RenderContacts(opts.IOStreams, rows)
}

// newFetch returns the production Fetch closure that calls the Telegram API.
func newFetch(f *runtime.Invocation) actioncontact.ListFunc {
	return func(ctx context.Context, q actioncontact.ListQuery) ([]output.ContactRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		var rows []output.ContactRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, pm *peers.Manager, res *peer.Resolver) error {
				r, ferr := telegram.ListContacts(ctx, api, pm, q)
				rows = r
				if ferr == nil {
					recordContactPeers(res.Store(), rows)
				}
				return ferr
			})
		return rows, err
	}
}

func recordContactPeers(store *account.PeerStore, rows []output.ContactRow) {
	if store == nil {
		return
	}
	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i]
		ref := contactRef(row)
		if ref == "" {
			continue
		}
		kind := "user"
		if row.Bot {
			kind = "bot"
		}
		_ = store.RecordRecentPeer(account.RecentPeer{
			Ref:      ref,
			ID:       row.ID,
			Kind:     kind,
			Title:    contactTitle(row),
			Username: row.Username,
		})
	}
}

func contactRef(row output.ContactRow) string {
	if row.Username != "" {
		return "@" + row.Username
	}
	if row.ID != 0 {
		return strconv.FormatInt(row.ID, 10)
	}
	return ""
}

func contactTitle(row output.ContactRow) string {
	return strings.TrimSpace(strings.Join([]string{row.FirstName, row.LastName}, " "))
}
