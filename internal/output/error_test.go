package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
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
