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

	"github.com/gotd/td/tg"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// WorkerOptions plumbs the live runtime into Run. The worker holds the
// account flock via session.Run's existing acquisition path (acct.Lock
// happens inside f.WithClient), so a second invocation while the
// daemon is running fails with account.ErrBusy.
type WorkerOptions struct {
	Inv       *runtime.Invocation
	Account   *account.Account
	IOStreams *ui.IOStreams
}

// Run is the entry point for the background worker that gets exec'd by
// the OS service manager (launchctl/systemctl/etc.) via the hidden
// "tg daemon run" subcommand. It:
//
//  1. acquires the account flock (transitively via WithClient)
//  2. opens a long-lived MTProto connection with an UpdateDispatcher
//  3. appends every update to UpdatesFile(account) as ndjson
//  4. waits for SIGTERM/SIGINT to exit cleanly
//
// Errors during the MTProto session are returned so the service
// manager's restart policy (KeepAlive on launchd, Restart=on-failure
// on systemd) takes over.
func Run(ctx context.Context, opts WorkerOptions) error {
	if opts.Inv == nil || opts.Account == nil {
		return errors.New("daemon worker requires Inv and Account")
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
	// Rotate any previous log eagerly so a long-lived service does not
	// grow unbounded. The actual writes go through stdout/stderr that
	// the service definition redirects; rotation here trims the file
	// at startup.
	_ = RotateIfLarger(LogFile(opts.Account.Meta.Name), DefaultLogMaxSize)

	sink, err := openUpdatesSink(UpdatesFile(opts.Account.Meta.Name))
	if err != nil {
		return err
	}
	defer func() { _ = sink.Close() }()

	events := make(chan telegram.WatchEvent, 128)
	disp := tg.NewUpdateDispatcher()
	telegram.RegisterWatchHandlers(disp, telegram.WatchFilter{}, nil, events)

	clientOpts := runtime.ClientOptsFrom(opts.Inv, opts.Account)
	clientOpts.UpdateHandler = disp

	return opts.Inv.WithClient(ctx, opts.Account, clientOpts,
		func(ctx context.Context, _ session.Client) error {
			fmt.Fprintf(os.Stderr, "tg daemon: connected for account %q, streaming to %s\n",
				opts.Account.Meta.Name, UpdatesFile(opts.Account.Meta.Name))
			return drainToSink(ctx, events, sink)
		})
}

// drainToSink pumps watch events into sink until ctx is done. The sink
// is its own writer rather than the worker's stdout so the service log
// (stderr / stdout) stays distinct from the structured event stream.
func drainToSink(ctx context.Context, events <-chan telegram.WatchEvent, sink *updatesSink) error {
	enc := json.NewEncoder(sink)
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-events:
			if err := enc.Encode(ev); err != nil {
				return fmt.Errorf("write update: %w", err)
			}
		}
	}
}

// updatesSink is a mutex-guarded append writer. The mutex is forward
// compatible with Phase 3, where the daemon will fan an event to the
// file and to N socket subscribers under the same lock.
type updatesSink struct {
	mu sync.Mutex
	f  *os.File
}

func openUpdatesSink(path string) (*updatesSink, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open updates sink: %w", err)
	}
	return &updatesSink{f: f}, nil
}

func (s *updatesSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.f.Write(p)
}

func (s *updatesSink) Close() error {
	if s == nil || s.f == nil {
		return nil
	}
	return s.f.Close()
}
