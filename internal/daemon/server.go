package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/vika2603/telegram-cli/internal/telegram"
)

// PeerRefResolver translates a raw peer reference ("@chan", "me",
// "c:NNN:H") into the normalized peer ID used by telegram.WatchFilter
// (peerID() in messages.go). The daemon supplies this via its live
// session so clients that lack MTProto access can still scope their
// subscription by ref.
type PeerRefResolver func(ctx context.Context, ref string) (int64, error)

// HandlerFunc is the contract for application-level RPC methods that
// are not connection-state-affecting (ping/subscribe/unsubscribe are
// built in). The worker registers one per CLI command it wants to
// route through the socket: me.show, msg.list, chat.resolve, etc.
//
// params is the raw JSON the client sent (may be empty). The return
// value is marshalled into Frame.Result on success or Frame.Error on
// failure — error messages should be safe to surface to scripts.
type HandlerFunc func(ctx context.Context, params json.RawMessage) (json.RawMessage, error)

// Server is the Unix-socket front door of the daemon. It accepts
// client connections, sends a Hello frame, dispatches incoming RPC
// frames to handlers, and fans out subscribed events back to the
// client. The dispatcher itself is owned by the worker; the server
// only consumes events via the SubscriptionManager.
type Server struct {
	account  string
	sock     string
	listen   net.Listener
	subs     *SubscriptionManager
	resolver PeerRefResolver
	metrics  *Metrics

	// handlers is the application-level RPC table. Methods not present
	// here fall through to the built-in handlers (ping/subscribe/
	// unsubscribe / daemon.stats) and finally to an "unknown_method"
	// error.
	handlersMu sync.RWMutex
	handlers   map[string]HandlerFunc

	// conns tracks live connections so Close can shut them down.
	connMu sync.Mutex
	conns  map[net.Conn]struct{}

	// closed is set on Close to short-circuit accept loops.
	closeOnce sync.Once
	closed    chan struct{}
}

// NewServer constructs a Server bound to sockPath. The caller is
// responsible for calling Serve in a goroutine; Close stops Serve and
// removes the socket file.
//
// account is recorded in the Hello frame so clients can confirm they
// are talking to the right daemon when multiple accounts are
// configured. subs is the producer side of the fanout — the worker
// hands the same instance to both gotd's dispatcher (via Publish) and
// here (so connections can Subscribe). resolver may be nil; when nil
// subscribe requests with Refs return an "unsupported" error.
func NewServer(account, sockPath string, subs *SubscriptionManager, resolver PeerRefResolver) *Server {
	return &Server{
		account:  account,
		sock:     sockPath,
		subs:     subs,
		resolver: resolver,
		metrics:  NewMetrics(),
		handlers: make(map[string]HandlerFunc),
		conns:    make(map[net.Conn]struct{}),
		closed:   make(chan struct{}),
	}
}

// Metrics returns the server's live metrics object. The same value is
// returned across calls, so tests and external watchers can attach
// before the server starts serving.
func (s *Server) Metrics() *Metrics { return s.metrics }

// Register binds an application-level RPC method to its handler. Call
// before Serve to avoid the race between Accept and method dispatch.
// Re-registering a name overwrites the previous handler — useful in
// tests, intentional in production where a worker reload may want to
// re-bind closures.
func (s *Server) Register(method string, h HandlerFunc) {
	s.handlersMu.Lock()
	defer s.handlersMu.Unlock()
	s.handlers[method] = h
}

func (s *Server) handler(method string) (HandlerFunc, bool) {
	s.handlersMu.RLock()
	defer s.handlersMu.RUnlock()
	h, ok := s.handlers[method]
	return h, ok
}

// Listen binds the Unix socket. Separate from Serve so the caller can
// surface bind errors before forking the goroutine.
func (s *Server) Listen() error {
	// Stale socket from a crashed prior daemon: remove and retry.
	if _, err := os.Stat(s.sock); err == nil {
		if rmErr := os.Remove(s.sock); rmErr != nil {
			return fmt.Errorf("remove stale socket: %w", rmErr)
		}
	}
	if err := EnsureDir(parentDir(s.sock)); err != nil {
		return err
	}
	ln, err := net.Listen("unix", s.sock)
	if err != nil {
		return fmt.Errorf("listen unix %s: %w", s.sock, err)
	}
	// Per-account daemon, per-user permissions. Reject other users.
	if err := os.Chmod(s.sock, 0o600); err != nil {
		_ = ln.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}
	s.listen = ln
	return nil
}

// Serve runs the accept loop until ctx is cancelled or Close is
// called. Each accepted connection runs in its own goroutine.
func (s *Server) Serve(ctx context.Context) error {
	if s.listen == nil {
		return errors.New("server.Serve called before Listen")
	}
	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()

	for {
		conn, err := s.listen.Accept()
		if err != nil {
			select {
			case <-s.closed:
				return nil
			default:
				if errors.Is(err, net.ErrClosed) {
					return nil
				}
				return fmt.Errorf("accept: %w", err)
			}
		}
		s.trackConn(conn)
		go s.handleConn(ctx, conn)
	}
}

// Close stops accepting new connections, closes all open connections,
// and removes the socket file. Safe to call multiple times.
func (s *Server) Close() error {
	var listenErr error
	s.closeOnce.Do(func() {
		close(s.closed)
		if s.listen != nil {
			listenErr = s.listen.Close()
		}
		s.connMu.Lock()
		for c := range s.conns {
			_ = c.Close()
		}
		s.conns = nil
		s.connMu.Unlock()
		_ = os.Remove(s.sock)
	})
	return listenErr
}

func (s *Server) trackConn(c net.Conn) {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	if s.conns == nil {
		_ = c.Close()
		return
	}
	s.conns[c] = struct{}{}
}

func (s *Server) untrackConn(c net.Conn) {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	if s.conns != nil {
		delete(s.conns, c)
	}
}

// handleConn is one goroutine per accepted connection. It owns a
// per-connection subscription set so we can clean them up when the
// client disconnects, and it serializes writes through a single
// json.Encoder (no concurrent writes from multiple goroutines).
func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer func() {
		_ = conn.Close()
		s.untrackConn(conn)
	}()

	enc := json.NewEncoder(conn)
	var encMu sync.Mutex
	writeFrame := func(f Frame) error {
		encMu.Lock()
		defer encMu.Unlock()
		return enc.Encode(f)
	}

	// Hello goes out first so clients can confirm account/schema before
	// sending requests.
	hello := mustMarshalEvent("hello", HelloPayload{
		DaemonVersion: "dev",
		Account:       s.account,
		Schema:        ProtocolSchema,
	})
	if err := writeFrame(hello); err != nil {
		return
	}

	// Per-connection subscription tracking: when conn drops, every
	// subscription it owns must go.
	connSubs := newConnSubSet()
	defer connSubs.closeAll(s.subs)

	scanner := bufio.NewScanner(conn)
	// Allow large frames — message text + entities can push past the
	// default 64 KB limit.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		var req Frame
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			_ = writeFrame(errorFrame(0, "invalid_frame", 64, err.Error()))
			continue
		}
		s.dispatch(ctx, &req, writeFrame, connSubs)
	}
}

// dispatch routes a single Request frame to the handler matching
// req.Method and writes the Response (or starts a goroutine fanning
// events to the client).
func (s *Server) dispatch(
	ctx context.Context,
	req *Frame,
	writeFrame func(Frame) error,
	connSubs *connSubSet,
) {
	switch req.Method {
	case "ping":
		_ = writeFrame(Frame{ID: req.ID, Result: rawJSON(`"pong"`)})

	case "daemon.stats":
		// Built-in introspection RPC. Kept here (not in the registry)
		// so it is always available even before registerHandlers runs
		// — useful for status checks that may race against worker
		// bootup.
		snap := s.metrics.Snapshot()
		snap.Subscriptions = int64(s.subs.Len())
		body, mErr := json.Marshal(snap)
		if mErr != nil {
			_ = writeFrame(errorFrame(req.ID, "internal", 1, mErr.Error()))
			return
		}
		_ = writeFrame(Frame{ID: req.ID, Result: body})

	case "subscribe":
		var params SubscribeParams
		if len(req.Params) > 0 {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				_ = writeFrame(errorFrame(req.ID, "invalid_params", 64, err.Error()))
				return
			}
		}
		filter, err := s.buildFilter(ctx, params)
		if err != nil {
			_ = writeFrame(errorFrame(req.ID, "resolve_failed", 74, err.Error()))
			return
		}
		sub := s.subs.Subscribe(filter)
		connSubs.add(sub.ID)
		result, marshalErr := json.Marshal(SubscribeResult{SubscriptionID: sub.ID})
		if marshalErr != nil {
			_ = writeFrame(errorFrame(req.ID, "internal", 1, marshalErr.Error()))
			return
		}
		_ = writeFrame(Frame{ID: req.ID, Result: result})
		go s.streamSubscription(ctx, sub, writeFrame)

	case "unsubscribe":
		var params UnsubscribeParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			_ = writeFrame(errorFrame(req.ID, "invalid_params", 64, err.Error()))
			return
		}
		s.subs.Unsubscribe(params.SubscriptionID)
		connSubs.remove(params.SubscriptionID)
		_ = writeFrame(Frame{ID: req.ID, Result: rawJSON("true")})

	default:
		if h, ok := s.handler(req.Method); ok {
			// Run the handler off the dispatch goroutine so a slow
			// method does not block the per-connection request stream.
			go func(req Frame, h HandlerFunc) {
				start := time.Now()
				result, err := h(ctx, req.Params)
				s.metrics.RecordRPC(req.Method, time.Since(start), err)
				if err != nil {
					_ = writeFrame(errorFrame(req.ID, "method_failed", 1, err.Error()))
					return
				}
				_ = writeFrame(Frame{ID: req.ID, Result: result})
			}(*req, h)
			return
		}
		_ = writeFrame(errorFrame(req.ID, "unknown_method", 64,
			fmt.Sprintf("unknown method %q", req.Method)))
	}
}

// streamSubscription forwards events from sub.C onto the connection
// as ndjson "update" frames. Drops surface as a single "lag" frame
// per overflow window so clients can resync.
func (s *Server) streamSubscription(
	ctx context.Context,
	sub *Subscription,
	writeFrame func(Frame) error,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-sub.C:
			if !ok {
				return
			}
			if dropped := sub.ResetDropped(); dropped > 0 {
				lag, marshalErr := json.Marshal(LagPayload{Dropped: dropped})
				if marshalErr != nil {
					return
				}
				if err := writeFrame(Frame{Event: "lag", Sub: sub.ID, Data: lag}); err != nil {
					return
				}
			}
			frame, err := MarshalUpdate(sub.ID, ev)
			if err != nil {
				return
			}
			if err := writeFrame(frame); err != nil {
				return
			}
		}
	}
}

// buildFilter materializes the on-the-wire SubscribeParams into the
// internal WatchFilter, resolving Refs through the server's resolver.
// Refs and PeerIDs union; an empty filter still matches everything.
func (s *Server) buildFilter(ctx context.Context, p SubscribeParams) (telegram.WatchFilter, error) {
	f := telegram.WatchFilter{}
	if len(p.Kinds) > 0 {
		f.Kinds = make(map[telegram.WatchEventKind]struct{}, len(p.Kinds))
		for _, k := range p.Kinds {
			f.Kinds[telegram.WatchEventKind(k)] = struct{}{}
		}
	}
	if len(p.PeerIDs) > 0 || len(p.Refs) > 0 {
		f.PeerIDs = make(map[int64]struct{}, len(p.PeerIDs)+len(p.Refs))
		for _, id := range p.PeerIDs {
			f.PeerIDs[id] = struct{}{}
		}
		if len(p.Refs) > 0 {
			if s.resolver == nil {
				return telegram.WatchFilter{}, errors.New("daemon has no peer resolver wired; pass peer_ids instead of refs")
			}
			for _, raw := range p.Refs {
				id, err := s.resolver(ctx, raw)
				if err != nil {
					return telegram.WatchFilter{}, fmt.Errorf("resolve %q: %w", raw, err)
				}
				f.PeerIDs[id] = struct{}{}
			}
		}
	}
	return f, nil
}

func errorFrame(id uint64, code string, exitCode int, msg string) Frame {
	return Frame{ID: id, Error: &FrameError{
		Code:     code,
		ExitCode: exitCode,
		Message:  msg,
	}}
}

func rawJSON(s string) json.RawMessage { return json.RawMessage(s) }

func mustMarshalEvent(name string, payload any) Frame {
	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return Frame{Event: name, Data: data}
}

func parentDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}

// connSubSet tracks the subscription IDs owned by a single connection
// so closing the connection can clean them up. Protected by its own
// mutex to keep handler dispatch lock-free apart from this small set.
type connSubSet struct {
	mu  sync.Mutex
	ids map[uint64]struct{}
}

func newConnSubSet() *connSubSet { return &connSubSet{ids: map[uint64]struct{}{}} }

func (c *connSubSet) add(id uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ids[id] = struct{}{}
}

func (c *connSubSet) remove(id uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.ids, id)
}

func (c *connSubSet) closeAll(mgr *SubscriptionManager) {
	c.mu.Lock()
	ids := make([]uint64, 0, len(c.ids))
	for id := range c.ids {
		ids = append(ids, id)
	}
	c.ids = nil
	c.mu.Unlock()
	for _, id := range ids {
		mgr.Unsubscribe(id)
	}
}

// Discard is the io.Writer that the bootstrap path uses when we are
// only interested in keeping a reader busy. Kept here so tests can
// share the singleton without importing io repeatedly.
var Discard io.Writer = io.Discard
