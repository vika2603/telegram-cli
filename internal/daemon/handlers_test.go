package daemon

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRegisterHandlers_BindsExpectedMethods verifies the full set of
// RPC method names that registerHandlers wires up. Catches typos and
// accidental removals — every CLI command that routes through the
// daemon depends on the matching name on this list.
func TestRegisterHandlers_BindsExpectedMethods(t *testing.T) {
	// Pass nils for api / pm / res / acct: registerHandlers only stores
	// closures; nothing invokes them in this test. We deliberately do
	// not call any handler so the captured nils never deref.
	srv := NewServer("test", "/tmp/should-not-be-used.sock",
		NewSubscriptionManager(1), nil)
	registerHandlers(srv, nil, nil, nil, nil)

	expected := []string{
		// Phase 4 reads
		"me.show",
		"chat.resolve",
		"chat.list",
		"msg.list",
		// Phase 5 writes
		"msg.send",
		"msg.edit",
		"msg.forward",
		"msg.delete",
		"msg.pin",
		"msg.react",
		// Phase 6 search / contact / profile
		"search.msg",
		"search.chat",
		"contact.list",
		"contact.add",
		"contact.delete",
		"contact.block",
		"contact.unblock",
		"profile.set_name",
		"profile.set_bio",
		"profile.set_username",
		"profile.set_status",
		"profile.delete_photo",
		// Phase 8 remaining chat + msg
		"chat.join",
		"chat.leave",
		"chat.mark_read",
		"chat.mute",
		"chat.unmute",
		"chat.folder",
		"msg.link",
		"msg.schedule_list",
		"msg.schedule_cancel",
	}

	for _, m := range expected {
		_, ok := srv.handler(m)
		require.True(t, ok, "method %q should be registered", m)
	}

	// Sanity: the count should match exactly. If new methods land,
	// expected[] must grow alongside.
	srv.handlersMu.RLock()
	got := len(srv.handlers)
	srv.handlersMu.RUnlock()
	require.Equal(t, len(expected), got,
		"handler count drift: expected %d, got %d", len(expected), got)
}

// TestRegisterMethod_Overwrites verifies the registry semantics
// callers depend on: re-registering a method swaps the handler
// without leaking stale closures.
func TestRegisterMethod_Overwrites(t *testing.T) {
	srv := NewServer("test", "/tmp/should-not-be-used.sock",
		NewSubscriptionManager(1), nil)

	srv.Register("x", func(context.Context, json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`1`), nil
	})
	srv.Register("x", func(context.Context, json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`2`), nil
	})

	h, ok := srv.handler("x")
	require.True(t, ok)
	out, err := h(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, "2", string(out))
}
