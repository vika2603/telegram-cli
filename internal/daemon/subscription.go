package daemon

import (
	"sync"
	"sync/atomic"

	"github.com/vika2603/telegram-cli/internal/telegram"
)

// Subscription is one client's view onto the daemon's update stream.
// The owner reads from C until the daemon closes it (on unsubscribe or
// connection drop). Dropped is incremented when the buffer is full and
// an event must be discarded; the server emits a "lag" frame in that
// case so the client knows to resync via a snapshot RPC (Phase 4).
type Subscription struct {
	ID      uint64
	Filter  telegram.WatchFilter
	C       chan telegram.WatchEvent
	dropped atomic.Uint64

	// closed guards C so concurrent Closes are safe.
	closeOnce sync.Once
}

// Close terminates the subscription's channel exactly once.
func (s *Subscription) Close() {
	s.closeOnce.Do(func() { close(s.C) })
}

// Dropped returns the running count of dropped events. Reset by the
// caller after they emit a "lag" notice.
func (s *Subscription) Dropped() uint64 {
	return s.dropped.Load()
}

// ResetDropped atomically clears the dropped counter and returns the
// previous value, used by the server when emitting "lag" frames.
func (s *Subscription) ResetDropped() uint64 {
	return s.dropped.Swap(0)
}

// SubscriptionManager owns all live subscriptions and is the fanout
// point between the gotd UpdateDispatcher (single producer) and N
// connected clients (multiple consumers). Calls are safe under
// concurrent Subscribe/Unsubscribe/Publish.
type SubscriptionManager struct {
	mu      sync.RWMutex
	nextID  uint64
	subs    map[uint64]*Subscription
	bufSize int
}

// NewSubscriptionManager constructs a manager with the given
// per-subscription buffer size. 64 is a reasonable default for a
// per-account daemon — bursty channel posts will not stall the
// dispatcher unless the client is genuinely backed up for seconds.
func NewSubscriptionManager(bufSize int) *SubscriptionManager {
	if bufSize <= 0 {
		bufSize = 64
	}
	return &SubscriptionManager{
		subs:    make(map[uint64]*Subscription),
		bufSize: bufSize,
	}
}

// Subscribe registers a new subscription with the given filter and
// returns the Subscription handle. The handle's ID is also the value
// passed back to clients in SubscribeResult.
func (m *SubscriptionManager) Subscribe(filter telegram.WatchFilter) *Subscription {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	sub := &Subscription{
		ID:     m.nextID,
		Filter: filter,
		C:      make(chan telegram.WatchEvent, m.bufSize),
	}
	m.subs[sub.ID] = sub
	return sub
}

// Unsubscribe removes the subscription by ID. Returns true if a
// subscription was found and removed, false if it was already gone.
// The subscription's channel is closed so the consumer goroutine
// exits cleanly.
func (m *SubscriptionManager) Unsubscribe(id uint64) bool {
	m.mu.Lock()
	sub, ok := m.subs[id]
	if ok {
		delete(m.subs, id)
	}
	m.mu.Unlock()
	if ok {
		sub.Close()
	}
	return ok
}

// Publish fans an event out to every matching subscription. The
// per-subscription send is non-blocking: if the buffer is full the
// event is dropped and Subscription.dropped is incremented, so a slow
// client cannot wedge the dispatcher.
func (m *SubscriptionManager) Publish(ev telegram.WatchEvent) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, sub := range m.subs {
		if !matches(sub.Filter, ev) {
			continue
		}
		select {
		case sub.C <- ev:
		default:
			sub.dropped.Add(1)
		}
	}
}

// Close terminates every subscription. Called when the daemon shuts
// down so connected clients see the channel close and exit cleanly.
func (m *SubscriptionManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, sub := range m.subs {
		sub.Close()
	}
	m.subs = nil
}

// Len reports the current live subscription count. Useful for
// status / metrics.
func (m *SubscriptionManager) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.subs)
}

// matches mirrors telegram.WatchFilter semantics. Duplicated here
// rather than reused via a method so we can inline the check under
// the manager's read lock without leaking the filter type's internals.
func matches(f telegram.WatchFilter, ev telegram.WatchEvent) bool {
	if len(f.Kinds) > 0 {
		if _, ok := f.Kinds[ev.Kind]; !ok {
			return false
		}
	}
	if len(f.PeerIDs) > 0 {
		// Single-row events: gate on Row.FromID.
		// Delete events carry Rows[]; accept if any one matches.
		if len(ev.Rows) > 0 {
			matched := false
			for _, r := range ev.Rows {
				if _, ok := f.PeerIDs[r.FromID]; ok {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		} else if _, ok := f.PeerIDs[ev.Row.FromID]; !ok {
			return false
		}
	}
	return true
}
