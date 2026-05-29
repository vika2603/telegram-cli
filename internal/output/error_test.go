package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gotd/td/tgerr"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
)

func TestEmitError_jsonShape(t *testing.T) {
	var buf bytes.Buffer
	wrapped := fmt.Errorf("resolve peer %q: %w", "@alice", peer.ErrForbidden)
	code := EmitError(&buf, "json", wrapped)
	require.Equal(t, 5, code)

	var obj struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		ExitCode int `json:"exit_code"`
	}
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &obj))
	require.Equal(t, "peer_forbidden", obj.Error.Code)
	require.Equal(t, 5, obj.ExitCode)
	require.Contains(t, obj.Error.Message, "@alice")
}

func TestEmitError_humanShape(t *testing.T) {
	var buf bytes.Buffer
	code := EmitError(&buf, "human", fmt.Errorf("boom: %w", command.ErrUsage))
	require.Equal(t, 2, code)
	require.Contains(t, buf.String(), "error: boom")
	require.False(t, json.Valid(buf.Bytes()), "human mode must not be valid JSON")
}

func TestEmitError_humanCancelIsQuiet(t *testing.T) {
	var buf bytes.Buffer
	// Ctrl+C mid-prompt surfaces as a wrapped command.ErrCancel; the human
	// path must exit 130 without printing a noisy "error: ..." line (the
	// terminal already showed "^C").
	code := EmitError(&buf, "human", fmt.Errorf("read api_id: %w", command.ErrCancel))
	require.Equal(t, 130, code)
	require.Empty(t, buf.String())
}

func TestEmitError_jsonCancelStillEmits(t *testing.T) {
	var buf bytes.Buffer
	code := EmitError(&buf, "json", fmt.Errorf("read api_id: %w", command.ErrCancel))
	require.Equal(t, 130, code)
	require.True(t, json.Valid(bytes.TrimSpace(buf.Bytes())), "json mode still emits a structured record")
}

// TestEmitError_floodWaitSurfacesRetryAfter guards the structured
// flood-wait detail: when a *session.FloodWaitError reaches EmitError
// in JSON mode, the seconds are surfaced as a typed
// `retry_after_seconds` int inside the `error` object so agents can
// sleep on it without regexing the message string.
func TestEmitError_floodWaitSurfacesRetryAfter(t *testing.T) {
	var buf bytes.Buffer
	// Wrap the typed error to simulate a real call path (commands always
	// add context via fmt.Errorf %w before returning).
	wrapped := fmt.Errorf("send to @alice: %w", &session.FloodWaitError{Seconds: 42})
	code := EmitError(&buf, "json", wrapped)
	require.Equal(t, 6, code)

	var obj struct {
		Error struct {
			Code              string `json:"code"`
			Message           string `json:"message"`
			RetryAfterSeconds int    `json:"retry_after_seconds"`
		} `json:"error"`
		ExitCode int `json:"exit_code"`
	}
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &obj))
	require.Equal(t, "flood_wait", obj.Error.Code)
	require.Equal(t, 42, obj.Error.RetryAfterSeconds)
	require.Equal(t, 6, obj.ExitCode)
}

// TestEmitError_rawGotdFloodWaitSurfacesRetryAfter exercises the
// fallback path that classifies raw *tgerr.Error FLOOD_WAITs (the
// shape gotd actually surfaces on every call path that doesn't go
// through ApplyFloodPolicy — which is most of them). Without this
// fallback the code would be "unknown" and retry_after_seconds
// would be absent.
func TestEmitError_rawGotdFloodWaitSurfacesRetryAfter(t *testing.T) {
	// Build a raw gotd FLOOD_WAIT exactly the way the server would.
	raw := tgerr.New(420, "FLOOD_WAIT_13")
	wrapped := fmt.Errorf("messages.search: %w", raw)

	var buf bytes.Buffer
	code := EmitError(&buf, "json", wrapped)
	require.Equal(t, 6, code, "raw gotd FLOOD_WAIT must map to exit 6")

	var obj struct {
		Error struct {
			Code              string `json:"code"`
			Message           string `json:"message"`
			RetryAfterSeconds int    `json:"retry_after_seconds"`
		} `json:"error"`
		ExitCode int `json:"exit_code"`
	}
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &obj))
	require.Equal(t, "flood_wait", obj.Error.Code, "must classify as flood_wait, not unknown")
	require.Equal(t, 13, obj.Error.RetryAfterSeconds, "must extract seconds from FLOOD_WAIT_<N>")
}

// TestEmitError_nonDetailerOmitsField asserts no spurious key on
// errors that don't implement ErrorDetailer.
func TestEmitError_nonDetailerOmitsField(t *testing.T) {
	var buf bytes.Buffer
	EmitError(&buf, "json", fmt.Errorf("boom: %w", command.ErrUsage))
	var raw map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &raw))
	errBlock := raw["error"].(map[string]any)
	require.NotContains(t, errBlock, "retry_after_seconds")
}

// TestEmitError_detailerCannotOverwriteCoreFields guards the merge
// guard in EmitError: a malicious / sloppy detailer returning "code"
// or "message" in its detail map must not clobber the authoritative
// fields from status.Code/err.Error.
func TestEmitError_detailerCannotOverwriteCoreFields(t *testing.T) {
	var buf bytes.Buffer
	EmitError(&buf, "json", &noisyDetailerError{})
	var raw map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &raw))
	errBlock := raw["error"].(map[string]any)
	require.NotEqual(t, "HIJACKED", errBlock["code"])
	require.NotEqual(t, "HIJACKED", errBlock["message"])
	require.Equal(t, "ok", errBlock["extra"])
}

type noisyDetailerError struct{}

func (noisyDetailerError) Error() string { return "real message" }
func (noisyDetailerError) ErrorDetail() map[string]any {
	return map[string]any{
		"code":    "HIJACKED",
		"message": "HIJACKED",
		"extra":   "ok",
	}
}
