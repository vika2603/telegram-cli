// Package watch implements the top-level "tg watch" command.
//
// Watch is the first CLI surface that holds a long-lived MTProto
// connection. Where every other command does dial → RPC → exit, watch
// runs telegram.Client.Run for the lifetime of the process and bridges
// the server-pushed UpdateDispatcher into stdout as ndjson.
//
// When a daemon is running for the active account, watch detects its
// socket and routes through the IPC subscription path instead of
// dialing MTProto itself. This avoids spinning a second MTProto
// session per `tg watch` invocation when the daemon already holds
// one — multiple sessions per account are legal but wasteful and
// share rate-limit budget. The local path is the fallback when no
// daemon is reachable or when --no-daemon is set.
package watch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/account"
	actionwatch "github.com/vika2603/telegram-cli/internal/action/watch"
	"github.com/vika2603/telegram-cli/internal/cli/complete"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/daemon"
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
	NoDaemon  bool
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
			opts.Stream = newStream(f, opts)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringSliceVar(&opts.Kinds, "kind", nil, "Filter event kinds (repeatable / comma-separated): message,edit,delete")
	cmd.Flags().IntVar(&opts.Limit, "limit", 0, "Exit after N events (0 = stream until cancelled)")
	cmd.Flags().BoolVar(&opts.NoDaemon, "no-daemon", false, "Force local MTProto connection even if a daemon is reachable")
	// Watch does NOT set NeedsClient: when a daemon is running the
	// client never dials MTProto, so the precondition would fail
	// gratuitously. The streaming path explicitly resolves the account
	// and chooses local vs. daemon mode.
	command.SetMeta(cmd, command.Meta{NeedsAccount: true})
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

// newStream picks daemon mode or local mode at run time. Daemon mode
// is preferred whenever the socket is reachable for the active
// account, because the daemon owns the flock and a local dial would
// fail anyway. --no-daemon forces the local path.
func newStream(f *runtime.Invocation, opts *Options) actionwatch.StreamFunc {
	return func(ctx context.Context, q actionwatch.Query) error {
		acct, err := f.Account("")
		if err != nil {
			return err
		}
		if !opts.NoDaemon && daemon.DaemonReachable(acct.Meta.Name) {
			return streamViaDaemon(ctx, acct.Meta.Name, opts.IOStreams, q)
		}
		return streamLocal(ctx, f, acct, opts.IOStreams, q)
	}
}

// streamViaDaemon connects to the per-account daemon socket and
// subscribes for the requested kinds + refs. The daemon resolves refs
// server-side using its live session, so the client never touches
// MTProto.
func streamViaDaemon(
	ctx context.Context,
	accountName string,
	ios *ui.IOStreams,
	q actionwatch.Query,
) error {
	cl, err := daemon.Dial(ctx, accountName)
	if err != nil {
		return fmt.Errorf("connect to daemon: %w", err)
	}
	defer func() { _ = cl.Close() }()

	params := daemon.SubscribeParams{Kinds: q.Kinds}
	for _, r := range q.Refs {
		params.Refs = append(params.Refs, r.String())
	}
	subID, err := cl.SubscribeRaw(ctx, params)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	emitted := 0
	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}
			return ctx.Err()
		case frame, ok := <-cl.Events:
			if !ok {
				return errors.New("daemon closed the connection")
			}
			if frame.Sub != subID || frame.Event != "update" {
				continue
			}
			if _, err := ios.Out.Write(append([]byte(frame.Data), '\n')); err != nil {
				return err
			}
			emitted++
			if q.Limit > 0 && emitted >= q.Limit {
				return nil
			}
		}
	}
}

// streamLocal is the original Phase-1 path: open a local MTProto
// session, register the dispatcher, write events to ios.Out. Used
// when no daemon is reachable (or when --no-daemon forces this path).
func streamLocal(
	ctx context.Context,
	f *runtime.Invocation,
	acct *account.Account,
	ios *ui.IOStreams,
	q actionwatch.Query,
) error {
	filter := telegram.WatchFilter{}
	if len(q.Refs) > 0 {
		filter.PeerIDs = make(map[int64]struct{}, len(q.Refs))
		err := f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, _ *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				for _, r := range q.Refs {
					resolved, err := res.Resolve(ctx, r)
					if err != nil {
						return err
					}
					filter.PeerIDs[telegram.NormalizeInputPeerID(resolved.InputPeer)] = struct{}{}
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

	events := make(chan telegram.WatchEvent, 64)
	disp := tg.NewUpdateDispatcher()
	telegram.RegisterWatchHandlers(disp, filter, nil, events)

	opts := runtime.ClientOptsFrom(f, acct)
	opts.UpdateHandler = disp

	return f.WithClient(ctx, acct, opts, func(ctx context.Context, _ session.Client) error {
		return streamLoop(ctx, ios, events, q.Limit)
	})
}

func streamLoop(ctx context.Context, ios *ui.IOStreams, events <-chan telegram.WatchEvent, limit int) error {
	enc := json.NewEncoder(ios.Out)
	emitted := 0
	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}
			return ctx.Err()
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
