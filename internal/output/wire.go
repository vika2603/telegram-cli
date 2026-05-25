package output

// Wire types for IPC transport.
//
// Several Row types (MessageRow, ChatRow, SearchMsgRow) carry custom
// MarshalJSON methods that produce a user-facing envelope shape
// (`{"from":{"id":...,"username":...,...}}`) different from the
// underlying struct fields (`FromID`, `FromUsername`, ...). They do
// not carry matching UnmarshalJSON, so a naive Marshal -> Unmarshal
// round trip drops every nested field.
//
// The daemon IPC layer needs round-trippable wire bytes — the client
// re-renders the user-facing shape on its own end via the same
// Exporter the local path uses. To get a round-trippable shape, the
// daemon marshals through these *Wire types: each is a `type X Y`
// definition (NOT a type alias), which keeps the field layout and
// json tags but strips MarshalJSON so encoding/json falls back to
// default field-tag based serialization. Unmarshal targets stay the
// original Row types — their default Unmarshal already matches the
// default Marshal of the wire types.

// MessageRowWire is the IPC-transport shape of MessageRow.
type MessageRowWire MessageRow

// ChatRowWire is the IPC-transport shape of ChatRow.
type ChatRowWire ChatRow

// SearchMsgRowWire is the IPC-transport shape of SearchMsgRow.
type SearchMsgRowWire SearchMsgRow

// MessageRowsToWire copies rows into the wire-shape slice. The cast
// is element-by-element because Go does not permit slice-type
// conversion between two named types with the same underlying
// element type.
func MessageRowsToWire(rows []MessageRow) []MessageRowWire {
	out := make([]MessageRowWire, len(rows))
	for i := range rows {
		out[i] = MessageRowWire(rows[i])
	}
	return out
}

// ChatRowsToWire is the slice variant for ChatRow.
func ChatRowsToWire(rows []ChatRow) []ChatRowWire {
	out := make([]ChatRowWire, len(rows))
	for i := range rows {
		out[i] = ChatRowWire(rows[i])
	}
	return out
}

// SearchMsgRowsToWire is the slice variant for SearchMsgRow.
func SearchMsgRowsToWire(rows []SearchMsgRow) []SearchMsgRowWire {
	out := make([]SearchMsgRowWire, len(rows))
	for i := range rows {
		out[i] = SearchMsgRowWire(rows[i])
	}
	return out
}
