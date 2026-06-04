package daemon

import (
	"encoding/json"

	"github.com/vika2603/telegram-cli/internal/telegram"
)

// ProtocolSchema is the wire-format version number daemon and client
// exchange in the Hello frame. Bump when a backward-incompatible field
// renames/removes happen; additive changes can stay at the same number
// because both ends ignore unknown JSON fields.
const ProtocolSchema = 1

// FeatureMediaSend is advertised in the Hello frame by daemons that can relay
// sticker/GIF sends over IPC (i.e. their SendQuery decoder understands the
// Sticker/Gif fields). A client that does not see this feature must fall back
// to the local path for media sends, because an older daemon would silently
// drop the unknown field and post an empty message. Additive on the wire:
// older daemons simply omit it, older clients ignore it.
const FeatureMediaSend = "msg.send.media"

// DaemonFeatures lists the optional capabilities this build advertises to
// clients in the Hello frame.
func DaemonFeatures() []string {
	return []string{FeatureMediaSend}
}

// Frame is the union sent in both directions over the Unix socket.
// Exactly one of Method / Result / Error / Event is populated. The
// daemon and client both decode into Frame first, then dispatch on
// which field is non-empty.
//
// Why a single union vs three top-level types: one MarshalJSON path,
// one Scanner per connection, and "I see something with id=X, route
// it" is easier than discriminating types.
type Frame struct {
	// ID correlates Request with Response. Set by the client on each
	// outbound Request; echoed verbatim on the daemon's Response. Zero
	// on server-initiated frames (Hello, Event).
	ID uint64 `json:"id,omitempty"`

	// Method names the RPC on outbound requests, e.g. "subscribe",
	// "unsubscribe", "ping". Empty on responses and events.
	Method string `json:"method,omitempty"`

	// Params is the method-specific payload, raw JSON so handlers can
	// type-assert into their own struct.
	Params json.RawMessage `json:"params,omitempty"`

	// Result is the success payload echoed back by the daemon. Mutually
	// exclusive with Error on a single Frame.
	Result json.RawMessage `json:"result,omitempty"`

	// Error carries the status code, exit code, and message on
	// failure.  The exit_code field mirrors internal/program/status so
	// clients can translate the RPC error into the same shell exit
	// they would have seen from a local invocation.
	Error *FrameError `json:"error,omitempty"`

	// Event names a server-pushed notification: "hello", "update",
	// "lag", "bye". Carries Data as the payload.
	Event string `json:"event,omitempty"`

	// Sub is the subscription_id this event belongs to. Zero for
	// connection-scoped events (hello, bye).
	Sub uint64 `json:"sub,omitempty"`

	// Data is the event payload.
	Data json.RawMessage `json:"data,omitempty"`
}

// FrameError is the JSON shape of an RPC error. Mirrors the status
// code + exit code contract the CLI surfaces from local commands.
// Detail is the structured per-error payload an output.ErrorDetailer
// would attach in the local path (e.g. retry_after_seconds for a
// FLOOD_WAIT). Carrying it through the IPC frame keeps the daemon
// path byte-equivalent to the local path on the wire.
type FrameError struct {
	Code     string         `json:"code"`
	ExitCode int            `json:"exit_code"`
	Message  string         `json:"message"`
	Detail   map[string]any `json:"detail,omitempty"`
}

// HelloPayload is the first frame the daemon pushes on every new
// connection. The client uses it to confirm the server is the right
// account and that the schema version matches.
type HelloPayload struct {
	DaemonVersion string   `json:"daemon_version"`
	Account       string   `json:"account"`
	Schema        int      `json:"schema"`
	Features      []string `json:"features,omitempty"`
}

// SubscribeParams names the kinds/peers a subscriber wants. Empty
// slices mean "all" (no filtering).
//
// Two ways to scope to specific peers, both optional:
//   - PeerIDs: pre-resolved numeric IDs (peerID() encoding, same as
//     telegram.WatchFilter.PeerIDs).
//   - Refs: raw peer references ("@chan", "me", "c:NNN:H", ...). The
//     daemon resolves them via its live session and unions the result
//     with PeerIDs. Required for clients that can't resolve themselves
//     because the daemon holds the account flock.
type SubscribeParams struct {
	Kinds   []string `json:"kinds,omitempty"`
	PeerIDs []int64  `json:"peer_ids,omitempty"`
	Refs    []string `json:"refs,omitempty"`
}

// SubscribeResult is the success response to a subscribe RPC.
type SubscribeResult struct {
	SubscriptionID uint64 `json:"subscription_id"`
}

// UnsubscribeParams cancels a previously created subscription.
type UnsubscribeParams struct {
	SubscriptionID uint64 `json:"subscription_id"`
}

// LagPayload is emitted on the connection when the per-subscription
// buffer overflows and one or more updates were dropped.
type LagPayload struct {
	Dropped uint64 `json:"dropped"`
}

// MarshalUpdate is a thin helper that wraps a telegram.WatchEvent in a
// Frame ready to be serialized. Keeping it here means the wire shape
// of the "update" event is owned by this package and not by callers.
func MarshalUpdate(sub uint64, ev telegram.WatchEvent) (Frame, error) {
	data, err := json.Marshal(ev)
	if err != nil {
		return Frame{}, err
	}
	return Frame{Event: "update", Sub: sub, Data: data}, nil
}
