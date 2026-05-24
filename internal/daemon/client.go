package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vika2603/telegram-cli/internal/telegram"
)

// Client is a thin wrapper around a Unix socket connection to a daemon.
// It owns the reader goroutine that demultiplexes responses (matched
// by request ID) and event streams (delivered on Events). The zero
// value is unusable; construct via Dial.
type Client struct {
	conn  net.Conn
	enc   *json.Encoder
	hello HelloPayload

	mu      sync.Mutex
	encMu   sync.Mutex // serializes Encode calls
	nextID  atomic.Uint64
	pending map[uint64]chan Frame

	// Events fires for every server-pushed event (update/lag/bye).
	// The channel is buffered so a slow consumer drops nothing on the
	// wire; reads should still be timely to avoid stalling the daemon's
	// per-subscription queue.
	Events chan Frame

	closeOnce sync.Once
	closed    chan struct{}
	readErr   atomic.Value // error
}

// DaemonReachable reports whether a daemon socket exists for account
// and accepts a connection. Used by client code (e.g. `tg watch`) to
// decide whether to route through the daemon or fall back to local
// MTProto. Errors are absorbed: any failure means "unreachable".
func DaemonReachable(account string) bool {
	path := SocketPath(account)
	info, err := os.Stat(path)
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		return false
	}
	conn, err := net.DialTimeout("unix", path, 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// Dial connects to the daemon for account and reads the Hello frame.
// Returns the live Client; the caller must Close it when done.
func Dial(ctx context.Context, account string) (*Client, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", SocketPath(account))
	if err != nil {
		return nil, fmt.Errorf("dial daemon: %w", err)
	}
	return AttachClient(conn)
}

// AttachClient wraps an already-established connection. It is the
// shared body of Dial and the seam tests use to connect via a custom
// socket path without rebuilding the SocketPath name resolution.
func AttachClient(conn net.Conn) (*Client, error) {
	c := &Client{
		conn:    conn,
		enc:     json.NewEncoder(conn),
		pending: make(map[uint64]chan Frame),
		Events:  make(chan Frame, 64),
		closed:  make(chan struct{}),
	}
	go c.readLoop()

	helloCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	select {
	case <-helloCtx.Done():
		_ = c.Close()
		return nil, errors.New("daemon did not send Hello within 3s")
	case f, ok := <-c.Events:
		if !ok {
			_ = c.Close()
			return nil, errors.New("daemon closed before Hello")
		}
		if f.Event != "hello" {
			_ = c.Close()
			return nil, fmt.Errorf("expected hello, got event %q", f.Event)
		}
		if err := json.Unmarshal(f.Data, &c.hello); err != nil {
			_ = c.Close()
			return nil, fmt.Errorf("decode hello: %w", err)
		}
		if c.hello.Schema != ProtocolSchema {
			_ = c.Close()
			return nil, fmt.Errorf("daemon schema %d, client expects %d",
				c.hello.Schema, ProtocolSchema)
		}
	}
	return c, nil
}

// Hello returns the welcome payload received at connect time.
func (c *Client) Hello() HelloPayload { return c.hello }

// Call sends a single RPC and waits for the matching response. params
// may be nil. Returns the raw Result; the caller unmarshals into the
// method-specific shape.
func (c *Client) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	var pBytes json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		pBytes = b
	}
	ch := make(chan Frame, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	if err := c.writeFrame(Frame{ID: id, Method: method, Params: pBytes}); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case f, ok := <-ch:
		if !ok {
			if rErr, _ := c.readErr.Load().(error); rErr != nil {
				return nil, rErr
			}
			return nil, errors.New("daemon closed connection")
		}
		if f.Error != nil {
			return nil, fmt.Errorf("%s: %s", f.Error.Code, f.Error.Message)
		}
		return f.Result, nil
	}
}

// Subscribe asks the daemon to start streaming updates matching filter.
// Returns the subscription ID; events flow onto c.Events tagged with
// Sub == this ID. Use SubscribeRaw to pass refs the daemon will resolve
// server-side (the cleaner path for `tg watch` since clients cannot
// dial MTProto themselves while the daemon holds the flock).
func (c *Client) Subscribe(ctx context.Context, filter telegram.WatchFilter) (uint64, error) {
	params := SubscribeParams{}
	if len(filter.Kinds) > 0 {
		for k := range filter.Kinds {
			params.Kinds = append(params.Kinds, string(k))
		}
	}
	if len(filter.PeerIDs) > 0 {
		for id := range filter.PeerIDs {
			params.PeerIDs = append(params.PeerIDs, id)
		}
	}
	return c.SubscribeRaw(ctx, params)
}

// SubscribeRaw is the wire-shape escape hatch for callers that have
// not pre-resolved their refs. The daemon resolves SubscribeParams.Refs
// server-side and unions them with PeerIDs.
func (c *Client) SubscribeRaw(ctx context.Context, params SubscribeParams) (uint64, error) {
	raw, err := c.Call(ctx, "subscribe", params)
	if err != nil {
		return 0, err
	}
	var out SubscribeResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return 0, err
	}
	return out.SubscriptionID, nil
}

// Unsubscribe cancels the given subscription. Events still in flight
// may arrive on c.Events before the server processes the cancel.
func (c *Client) Unsubscribe(ctx context.Context, id uint64) error {
	_, err := c.Call(ctx, "unsubscribe", UnsubscribeParams{SubscriptionID: id})
	return err
}

// Close ends the connection. Any in-flight Call returns
// "connection closed".
func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.closed)
		err = c.conn.Close()
		c.mu.Lock()
		for _, ch := range c.pending {
			close(ch)
		}
		c.pending = nil
		c.mu.Unlock()
		close(c.Events)
	})
	return err
}

// writeFrame is the only path that touches the encoder, guarded by a
// mutex because Subscribe / Call may both fire concurrently.
func (c *Client) writeFrame(f Frame) error {
	c.encMu.Lock()
	defer c.encMu.Unlock()
	return c.enc.Encode(f)
}

// readLoop consumes ndjson frames from the daemon, demultiplexing
// responses (by ID) and events (push to c.Events). On read error or
// EOF the connection is closed and all pending channels close.
func (c *Client) readLoop() {
	scanner := bufio.NewScanner(c.conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var f Frame
		if err := json.Unmarshal(scanner.Bytes(), &f); err != nil {
			c.readErr.Store(err)
			break
		}
		if f.Event != "" {
			select {
			case <-c.closed:
				return
			case c.Events <- f:
			}
			continue
		}
		c.mu.Lock()
		ch, ok := c.pending[f.ID]
		c.mu.Unlock()
		if ok {
			ch <- f
		}
	}
	if err := scanner.Err(); err != nil {
		c.readErr.Store(err)
	}
	_ = c.Close()
}
