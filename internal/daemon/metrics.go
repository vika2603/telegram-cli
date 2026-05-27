package daemon

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics is the daemon's lightweight observability surface. Counters
// are atomic; the per-method map is guarded by a small RWMutex so the
// vast majority of writes never contend. Snapshot is the only path
// that allocates, and it is called once per `tg daemon status` request.
type Metrics struct {
	startedAt time.Time

	updatesReceived atomic.Uint64
	subscriptions   atomic.Int64 // live gauge

	mu       sync.RWMutex
	rpcCalls map[string]*rpcMethodMetric
}

type rpcMethodMetric struct {
	calls      atomic.Uint64
	errors     atomic.Uint64
	totalNanos atomic.Uint64
}

// NewMetrics returns a Metrics initialized to the current wall clock.
// Callers wire this into Server and SubscriptionManager so writes
// happen on the hot path; reads (Snapshot) are on-demand.
func NewMetrics() *Metrics {
	return &Metrics{
		startedAt: time.Now().UTC(),
		rpcCalls:  make(map[string]*rpcMethodMetric),
	}
}

// IncUpdates is called once per dispatcher event.
func (m *Metrics) IncUpdates() {
	if m == nil {
		return
	}
	m.updatesReceived.Add(1)
}

// SetSubscriptions updates the live subscription count. The
// SubscriptionManager handles deltas; this method takes the absolute
// value so callers cannot underflow.
func (m *Metrics) SetSubscriptions(n int64) {
	if m == nil {
		return
	}
	m.subscriptions.Store(n)
}

// RecordRPC tallies a single application-method invocation. Pass
// err = nil for success; latency is measured by the caller because
// it owns the timing boundary.
func (m *Metrics) RecordRPC(method string, latency time.Duration, err error) {
	if m == nil {
		return
	}
	entry := m.entryFor(method)
	entry.calls.Add(1)
	if latency > 0 {
		entry.totalNanos.Add(uint64(latency.Nanoseconds()))
	}
	if err != nil {
		entry.errors.Add(1)
	}
}

func (m *Metrics) entryFor(method string) *rpcMethodMetric {
	m.mu.RLock()
	entry, ok := m.rpcCalls[method]
	m.mu.RUnlock()
	if ok {
		return entry
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry, ok := m.rpcCalls[method]; ok {
		return entry
	}
	entry = &rpcMethodMetric{}
	m.rpcCalls[method] = entry
	return entry
}

// MetricsSnapshot is the JSON-friendly view emitted by the daemon.stats
// RPC. Per-method maps are flattened so jq paths stay short.
type MetricsSnapshot struct {
	StartedAt       string                       `json:"started_at"`
	UptimeSeconds   float64                      `json:"uptime_seconds"`
	UpdatesReceived uint64                       `json:"updates_received"`
	Subscriptions   int64                        `json:"subscriptions"`
	RPCCalls        map[string]RPCMetricSnapshot `json:"rpc_calls,omitempty"`
}

// RPCMetricSnapshot is one entry of MetricsSnapshot.RPCCalls.
type RPCMetricSnapshot struct {
	Calls        uint64  `json:"calls"`
	Errors       uint64  `json:"errors"`
	AvgLatencyMs float64 `json:"avg_latency_ms,omitempty"`
}

// Snapshot is the read path. Allocates a fresh map so the caller can
// retain it without holding the mutex.
func (m *Metrics) Snapshot() MetricsSnapshot {
	if m == nil {
		return MetricsSnapshot{}
	}
	now := time.Now().UTC()
	snap := MetricsSnapshot{
		StartedAt:       m.startedAt.Format(time.RFC3339),
		UptimeSeconds:   now.Sub(m.startedAt).Seconds(),
		UpdatesReceived: m.updatesReceived.Load(),
		Subscriptions:   m.subscriptions.Load(),
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.rpcCalls) == 0 {
		return snap
	}
	snap.RPCCalls = make(map[string]RPCMetricSnapshot, len(m.rpcCalls))
	for name, entry := range m.rpcCalls {
		calls := entry.calls.Load()
		errs := entry.errors.Load()
		totalNs := entry.totalNanos.Load()
		var avgMs float64
		if calls > 0 && totalNs > 0 {
			avgMs = float64(totalNs) / float64(calls) / float64(time.Millisecond)
		}
		snap.RPCCalls[name] = RPCMetricSnapshot{
			Calls:        calls,
			Errors:       errs,
			AvgLatencyMs: avgMs,
		}
	}
	return snap
}
