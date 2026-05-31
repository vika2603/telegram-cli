package topic

import (
	"context"
	"fmt"
	"strconv"

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

// EditOptions holds the resolved flags and injected dependencies for the
// edit run.
type EditOptions struct {
	RawRef    string
	TopicID   int
	Title     string
	Close     bool
	Reopen    bool
	Hide      bool
	Unhide    bool
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Edit      actionchat.EditTopicFunc
}

func newEdit(f *runtime.Invocation, runF func(*EditOptions) error) *cobra.Command {
	opts := &EditOptions{}
	cmd := &cobra.Command{
		Use:               "edit <ref> <topic-id>",
		Short:             "Edit a forum topic (title, close, hide)",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			id, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("%w: topic id must be a positive integer", command.ErrUsage)
			}
			opts.TopicID = id
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Edit = newEditFn(f)
			return runEdit(cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Title, "title", "", "New title for the topic")
	cmd.Flags().BoolVar(&opts.Close, "close", false, "Close the topic")
	cmd.Flags().BoolVar(&opts.Reopen, "reopen", false, "Reopen a closed topic")
	cmd.Flags().BoolVar(&opts.Hide, "hide", false, "Hide the General topic")
	cmd.Flags().BoolVar(&opts.Unhide, "unhide", false, "Unhide the General topic")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"id", "title", "closed", "hidden"})
	return cmd
}

func runEdit(cmd *cobra.Command, opts *EditOptions) error {
	req := actionchat.EditTopicRequest{
		RawRef:  opts.RawRef,
		TopicID: opts.TopicID,
	}
	if cmd.Flags().Changed("title") {
		title := opts.Title
		req.Title = &title
	}
	if opts.Close && opts.Reopen {
		return fmt.Errorf("%w: --close and --reopen are mutually exclusive", command.ErrUsage)
	}
	if opts.Hide && opts.Unhide {
		return fmt.Errorf("%w: --hide and --unhide are mutually exclusive", command.ErrUsage)
	}
	if opts.Close {
		v := true
		req.Closed = &v
	} else if opts.Reopen {
		v := false
		req.Closed = &v
	}
	if opts.Hide {
		v := true
		req.Hidden = &v
	} else if opts.Unhide {
		v := false
		req.Hidden = &v
	}
	row, err := actionchat.EditTopic(cmd.Context(), req, opts.Edit)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderTopic(opts.IOStreams, row)
}

func newEditFn(f *runtime.Invocation) actionchat.EditTopicFunc {
	return func(ctx context.Context, q actionchat.EditTopicQuery) (output.TopicRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.TopicRow{}, err
		}
		var row output.TopicRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.EditForumTopic(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
