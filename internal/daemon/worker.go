package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/ref"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
)

// WithPeersFunc is the narrow surface this worker needs from
// runtime.Invocation. Declared locally so the daemon package does
// not have to import internal/runtime — that direction is reserved
// for internal/runtime importing daemon (for MaybeDial). Anyone with
// a runtime.Invocation can pass inv.WithPeers directly.
type WithPeersFunc func(
	ctx context.Context,
	acct *account.Account,
	opts session.Options,
	fn func(ctx context.Context, api *tg.Client, pm *peers.Manager, res *peer.Resolver) error,
) error

// WorkerOptions plumbs the live runtime into Run. Note the project's
// account.lock is **not** held for the daemon's lifetime; it is only
// taken briefly during atomic writes to account meta/session. Multiple
// MTProto sessions per account are legal (Telegram allows it), so a
// short-lived `tg send` while the daemon is running coexists fine.
type WorkerOptions struct {
	Account    *account.Account
	WithPeers  WithPeersFunc
	ClientOpts session.Options
}

// Run is the entry point for the background worker that gets exec'd by
// the OS service manager (launchctl/systemctl/etc.) via the hidden
// "tg daemon run" subcommand. It:
//
//  1. opens a long-lived MTProto connection with an UpdateDispatcher
//  2. appends every update to UpdatesFile(account) as ndjson, with
//     in-process size-based rotation
//  3. opens a Unix socket and fans live events to connected clients
//  4. waits for SIGTERM/SIGINT to exit cleanly
//
// Errors during the MTProto session are returned so the service
// manager's restart policy (KeepAlive on launchd, Restart=on-failure
// on systemd) takes over.
//
// Why WithPeers and not WithClient: clients connecting over the IPC
// socket may pass raw peer refs ("@chan", "me", ...) that the daemon
// must resolve on their behalf. session.directClient.ResolvePeer is
// stubbed out — only the peer.Resolver built by WithPeers can do real
// resolution via gotd's peers.Manager.
func Run(ctx context.Context, opts WorkerOptions) error {
	if opts.Account == nil || opts.WithPeers == nil {
		return errors.New("daemon worker requires Account and WithPeers")
	}

	// Honor SIGTERM/SIGINT so launchctl bootout / systemctl stop close
	// the MTProto session cleanly.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigs)
	go func() {
		select {
		case <-ctx.Done():
		case <-sigs:
			cancel()
		}
	}()

	if err := EnsureDir(DaemonDir(opts.Account.Meta.Name)); err != nil {
		return fmt.Errorf("ensure daemon dir: %w", err)
	}
	_ = RotateIfLarger(LogFile(opts.Account.Meta.Name), DefaultLogMaxSize)

	sink, err := openUpdatesSink(UpdatesFile(opts.Account.Meta.Name), DefaultLogMaxSize)
	if err != nil {
		return err
	}
	defer func() { _ = sink.Close() }()

	subs := NewSubscriptionManager(128)
	defer subs.Close()

	events := make(chan telegram.WatchEvent, 128)
	disp := tg.NewUpdateDispatcher()
	telegram.RegisterWatchHandlers(disp, telegram.WatchFilter{}, nil, events)

	clientOpts := opts.ClientOpts
	clientOpts.UpdateHandler = disp

	return opts.WithPeers(ctx, opts.Account, clientOpts,
		func(ctx context.Context, api *tg.Client, pm *peers.Manager, res *peer.Resolver) error {
			// Wrap the peer.Resolver into the simpler signature the
			// socket server speaks: raw ref string → normalized peer ID.
			resolver := func(ctx context.Context, raw string) (int64, error) {
				r, perr := ref.ParseRef(raw)
				if perr != nil {
					return 0, perr
				}
				resolved, rerr := res.Resolve(ctx, r)
				if rerr != nil {
					return 0, rerr
				}
				return telegram.NormalizeInputPeerID(resolved.InputPeer), nil
			}

			srv := NewServer(opts.Account.Meta.Name,
				SocketPath(opts.Account.Meta.Name), subs, resolver)
			registerHandlers(srv, opts.Account, api, pm, res)
			if err := srv.Listen(); err != nil {
				return fmt.Errorf("ipc server listen: %w", err)
			}
			defer func() { _ = srv.Close() }()

			fmt.Fprintf(os.Stderr,
				"tg daemon: connected for account %q\n  updates: %s\n  socket:  %s\n",
				opts.Account.Meta.Name,
				UpdatesFile(opts.Account.Meta.Name),
				SocketPath(opts.Account.Meta.Name))

			serverErr := make(chan error, 1)
			go func() { serverErr <- srv.Serve(ctx) }()

			pumpErr := pumpEvents(ctx, events, sink, subs, srv.Metrics())

			_ = srv.Close()
			<-serverErr
			return pumpErr
		})
}

// pumpEvents is the single consumer of the dispatcher channel. Every
// event is written to the on-disk sink (so the daemon still works as
// a tailable file even with no clients connected), published to the
// subscription manager (so live clients receive it), and counted in
// the daemon's metrics for the `tg daemon stats` RPC.
func pumpEvents(
	ctx context.Context,
	events <-chan telegram.WatchEvent,
	sink *updatesSink,
	subs *SubscriptionManager,
	metrics *Metrics,
) error {
	enc := json.NewEncoder(sink)
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-events:
			if err := enc.Encode(ev); err != nil {
				return fmt.Errorf("write update: %w", err)
			}
			subs.Publish(ev)
			metrics.IncUpdates()
		}
	}
}

// updatesSink is the mutex-guarded append writer for updates.ndjson.
// It tracks the current file size and rotates to `<path>.1` once
// maxSize bytes have accumulated, so a long-running daemon does not
// grow the file unbounded. maxSize <= 0 disables rotation.
type updatesSink struct {
	mu      sync.Mutex
	path    string
	maxSize int64
	f       *os.File
	written int64
}

func openUpdatesSink(path string, maxSize int64) (*updatesSink, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open updates sink: %w", err)
	}
	// Pick up the current size so a restart picks rotation right
	// where the prior worker left off (instead of resetting to 0
	// and accidentally letting the file grow far past maxSize).
	var size int64
	if info, statErr := f.Stat(); statErr == nil {
		size = info.Size()
	}
	return &updatesSink{path: path, maxSize: maxSize, f: f, written: size}, nil
}

func (s *updatesSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, err := s.f.Write(p)
	if err != nil {
		return n, err
	}
	s.written += int64(n)
	if s.maxSize > 0 && s.written >= s.maxSize {
		if rotErr := s.rotateLocked(); rotErr != nil {
			// Rotation failure should not break the daemon's main
			// loop; the write itself succeeded. Surface via the
			// returned error so the worker's stderr log shows it.
			return n, fmt.Errorf("rotate updates sink: %w", rotErr)
		}
	}
	return n, nil
}

// rotateLocked is called with s.mu held. It closes the current file,
// renames it to .1 (overwriting any prior backup), and opens a fresh
// file at the original path.
func (s *updatesSink) rotateLocked() error {
	if err := s.f.Close(); err != nil {
		return err
	}
	backup := s.path + ".1"
	_ = os.Remove(backup)
	if err := os.Rename(s.path, backup); err != nil {
		// Reopen the original file even if rename failed so the
		// next Write does not silently drop on a nil file.
		s.f, _ = os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		return err
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	s.f = f
	s.written = 0
	return nil
}

func (s *updatesSink) Close() error {
	if s == nil || s.f == nil {
		return nil
	}
	return s.f.Close()
}
