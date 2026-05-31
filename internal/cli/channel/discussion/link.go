package discussion

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

// Options holds the resolved flags and injected dependencies for link/unlink.
type Options struct {
	RawChannel string
	RawGroup   string
	Unlink     bool
	Exporter   output.Exporter
	IOStreams  *ui.IOStreams
	Do         actionchat.DiscussionFunc
}

func newLink(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:               "link <channel> <group>",
		Short:             "Link a supergroup as the channel's discussion group",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawChannel = args[0]
			opts.RawGroup = args[1]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Do = newDo(f)
			return Run(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "channel", "group"})
	return cmd
}

func newUnlink(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{Unlink: true}
	cmd := &cobra.Command{
		Use:               "unlink <channel>",
		Short:             "Unlink the channel's discussion group",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawChannel = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Do = newDo(f)
			return Run(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "channel", "group"})
	return cmd
}

// Run validates and dispatches the link/unlink operation.
func Run(ctx context.Context, opts *Options) error {
	req := actionchat.DiscussionRequest{RawChannel: opts.RawChannel, RawGroup: opts.RawGroup}
	var (
		row output.DiscussionRow
		err error
	)
	if opts.Unlink {
		row, err = actionchat.UnlinkDiscussion(ctx, req, opts.Do)
	} else {
		row, err = actionchat.LinkDiscussion(ctx, req, opts.Do)
	}
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	if row.Action == "link" {
		_, err = fmt.Fprintf(opts.IOStreams.Out, "linked %s → %s\n", peerName(row.Channel), peerName(*row.Group))
	} else {
		_, err = fmt.Fprintf(opts.IOStreams.Out, "unlinked discussion group from %s\n", peerName(row.Channel))
	}
	return err
}

func peerName(p output.PeerRef) string {
	switch {
	case p.Username != "":
		return "@" + p.Username
	case p.Title != "":
		return p.Title
	default:
		return p.Ref
	}
}

func newDo(f *runtime.Invocation) actionchat.DiscussionFunc {
	return func(ctx context.Context, q actionchat.DiscussionQuery) (output.DiscussionRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.DiscussionRow{}, err
		}
		var row output.DiscussionRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.SetChannelDiscussion(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
