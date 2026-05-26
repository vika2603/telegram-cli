package output

// Wire types for IPC transport.
//
// Several Row types (MessageRow, ChatRow, SearchChatRow) carry custom
// MarshalJSON methods that produce a user-facing envelope shape
// (`{"peer":{"id":..., "username":..., ...}}` or
// `{"from":{"id":...,"username":...,...}}`) different from the
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
//
// Row types **without** a custom MarshalJSON (UserRow, ContactRow,
// ProfileRow, SendResultRow, SearchMsgRow, ChatMembershipRow,
// ChatMuteRow, ChatFolderRow, ScheduledMessageRow, ...) marshal and
// unmarshal symmetrically by default and need no wire wrapper.

// MessageRowWire is the IPC-transport shape of MessageRow.
type MessageRowWire MessageRow

// ChatRowWire is the IPC-transport shape of ChatRow.
type ChatRowWire ChatRow

// SearchChatRowWire is the IPC-transport shape of SearchChatRow. Note
// SearchChatRow EMBEDS ChatRow; a naive `type SearchChatRowWire
// SearchChatRow` would strip SearchChatRow.MarshalJSON but still
// promote ChatRow.MarshalJSON via the embedded field, leaving the
// wire shape stuck in the envelope form. Embed ChatRowWire here
// instead so neither MarshalJSON applies.
type SearchChatRowWire struct {
	ChatRowWire
	Source string `json:"source,omitempty"`
}

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

// SearchChatRowsToWire is the slice variant for SearchChatRow. The
// cast cannot be a single type conversion because the wire shape
// embeds ChatRowWire (not ChatRow), so build each element explicitly.
func SearchChatRowsToWire(rows []SearchChatRow) []SearchChatRowWire {
	out := make([]SearchChatRowWire, len(rows))
	for i, r := range rows {
		out[i] = SearchChatRowWire{
			ChatRowWire: ChatRowWire(r.ChatRow),
			Source:      r.Source,
		}
	}
	return out
}
