package join

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/cli/complete"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// DecideOptions holds flags/deps for `join approve` / `deny`.
type DecideOptions struct {
	RawRef    string
	RawUsers  []string
	All       bool
	Link      string
	Approved  bool
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Do        actionchat.JoinDecisionFunc
}

func newApprove(f *runtime.Invocation, runF func(*DecideOptions) error) *cobra.Command {
	return decideCmd(f, runF, true, "approve", "Approve join request(s): one or more users, or --all")
}

func newDeny(f *runtime.Invocation, runF func(*DecideOptions) error) *cobra.Command {
	return decideCmd(f, runF, false, "deny", "Reject join request(s): one or more users, or --all")
}

func decideCmd(f *runtime.Invocation, runF func(*DecideOptions) error, approved bool, use, short string) *cobra.Command {
	opts := &DecideOptions{Approved: approved}
	cmd := &cobra.Command{
		Use:               use + " <ref> [user...]",
		Short:             short,
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.RawUsers = args[1:]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Do = newDecideFn(f)
			return runDecide(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.All, "all", false, "Act on every pending request (optionally scoped by --link)")
	cmd.Flags().StringVar(&opts.Link, "link", "", "With --all, only requests from this invite link")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "all", "peer", "error"})
	return cmd
}

func runDecide(ctx context.Context, opts *DecideOptions) error {
	req := actionchat.JoinDecisionRequest{
		RawRef:   opts.RawRef,
		RawUsers: opts.RawUsers,
		All:      opts.All,
		Link:     opts.Link,
	}
	var (
		rows []output.JoinResultRow
		err  error
	)
	if opts.Approved {
		rows, err = actionchat.ApproveJoin(ctx, req, opts.Do)
	} else {
		rows, err = actionchat.DenyJoin(ctx, req, opts.Do)
	}
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	verb := "approved"
	if !opts.Approved {
		verb = "denied"
	}
	for _, r := range rows {
		switch {
		case r.All:
			_, err = fmt.Fprintf(opts.IOStreams.Out, "%s all pending requests\n", verb)
		case r.Error != "":
			_, err = fmt.Fprintf(opts.IOStreams.Out, "failed %s: %s\n", peerName(r.Peer), r.Error)
		default:
			_, err = fmt.Fprintf(opts.IOStreams.Out, "%s %s\n", verb, peerName(r.Peer))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func peerName(p *output.PeerRef) string {
	if p == nil {
		return "?"
	}
	switch {
	case p.Username != "":
		return "@" + p.Username
	case p.Title != "":
		return p.Title
	default:
		return p.Ref
	}
}

func newDecideFn(f *runtime.Invocation) actionchat.JoinDecisionFunc {
	return func(ctx context.Context, q actionchat.JoinDecisionQuery) ([]output.JoinResultRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		var rows []output.JoinResultRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				rows, err = telegram.DecideJoinRequest(ctx, api, res, q)
				return err
			})
		return rows, err
	}
}
