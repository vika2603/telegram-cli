// Package watch implements the top-level "tg watch" command.
//
// Watch is the first CLI surface that holds a long-lived MTProto
// connection. Where every other command does dial → RPC → exit, watch
// runs telegram.Client.Run for the lifetime of the process and bridges
// the server-pushed UpdateDispatcher into stdout as ndjson. The
// foreground mode is the baseline before any daemon work — once the
// daemon arrives, this command will detect its socket and become a
// subscription client instead, with the foreground path kept as a
// fallback for unconfigured environments.
package watch

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionwatch "github.com/vika2603/telegram-cli/internal/action/watch"
	"github.com/vika2603/telegram-cli/internal/cli/complete"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	RawRefs   []string
	Kinds     []string
	Limit     int
	IOStreams *ui.IOStreams
	Stream    actionwatch.StreamFunc
}

// New builds the cobra command for "tg watch".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:               "watch [<ref>...]",
		Short:             "Stream real-time messages and edits as ndjson",
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRefs = args
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Stream = newStream(f, opts.IOStreams)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringSliceVar(&opts.Kinds, "kind", nil, "Filter event kinds (repeatable / comma-separated): message,edit,delete")
	cmd.Flags().IntVar(&opts.Limit, "limit", 0, "Exit after N events (0 = stream until cancelled)")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	return cmd
}

// Run validates the request and delegates to opts.Stream.
func Run(ctx context.Context, opts *Options) error {
	return actionwatch.Run(ctx, actionwatch.Request{
		RawRefs: opts.RawRefs,
		Kinds:   opts.Kinds,
		Limit:   opts.Limit,
	}, opts.Stream)
}

// newStream wires the production streaming path: open a long-lived
// MTProto session, register the UpdateDispatcher, write events to
// ios.Out as ndjson until ctx is cancelled or limit is reached.
func newStream(f *runtime.Invocation, ios *ui.IOStreams) actionwatch.StreamFunc {
	return func(ctx context.Context, q actionwatch.Query) error {
		acct, err := f.Account("")
		if err != nil {
			return err
		}

		// Pre-resolve refs to a peer-ID filter so the dispatcher can
		// drop unrelated traffic before we serialize anything.
		filter := telegram.WatchFilter{}
		if len(q.Refs) > 0 {
			filter.PeerIDs = make(map[int64]struct{}, len(q.Refs))
			err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
				func(ctx context.Context, _ *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
					for _, r := range q.Refs {
						resolved, err := res.Resolve(ctx, r)
						if err != nil {
							return err
						}
						filter.PeerIDs[normalizedID(resolved.InputPeer)] = struct{}{}
					}
					return nil
				})
			if err != nil {
				return err
			}
		}
		if len(q.Kinds) > 0 {
			filter.Kinds = make(map[telegram.WatchEventKind]struct{}, len(q.Kinds))
			for _, k := range q.Kinds {
				filter.Kinds[telegram.WatchEventKind(k)] = struct{}{}
			}
		}

		// Buffered to give the dispatcher headroom; if the consumer
		// stalls long enough to fill the buffer, gotd's handler will
		// block, which is the correct back-pressure signal.
		events := make(chan telegram.WatchEvent, 64)
		disp := tg.NewUpdateDispatcher()
		telegram.RegisterWatchHandlers(disp, filter, nil, events)

		opts := runtime.ClientOptsFrom(f, acct)
		opts.UpdateHandler = disp

		return f.WithClient(ctx, acct, opts, func(ctx context.Context, _ session.Client) error {
			return streamLoop(ctx, ios, events, q.Limit)
		})
	}
}

func streamLoop(ctx context.Context, ios *ui.IOStreams, events <-chan telegram.WatchEvent, limit int) error {
	enc := json.NewEncoder(ios.Out)
	emitted := 0
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if err == context.Canceled {
				return nil
			}
			return err
		case ev := <-events:
			if err := enc.Encode(ev); err != nil {
				return fmt.Errorf("encode watch event: %w", err)
			}
			emitted++
			if limit > 0 && emitted >= limit {
				return nil
			}
		}
	}
}

// normalizedID converts an InputPeer back to the same key scheme
// peerID() in telegram/messages.go produces, so dispatcher filtering
// matches MessageRow.FromID. Unrecognized variants return 0, which
// silently fails the filter — that is the correct behavior for refs
// we never resolved.
func normalizedID(p tg.InputPeerClass) int64 {
	switch v := p.(type) {
	case *tg.InputPeerUser:
		return v.UserID
	case *tg.InputPeerChat:
		return -v.ChatID
	case *tg.InputPeerChannel:
		return -1_000_000_000_000 - v.ChannelID
	case *tg.InputPeerSelf:
		return 0 // self only matches when filter is empty
	}
	return 0
}
