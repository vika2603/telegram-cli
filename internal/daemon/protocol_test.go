package daemon_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/daemon"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/telegram"
)

func TestFrame_RoundTripRequestResponse(t *testing.T) {
	req := daemon.Frame{
		ID:     7,
		Method: "subscribe",
		Params: json.RawMessage(`{"kinds":["message"]}`),
	}
	wire, err := json.Marshal(req)
	require.NoError(t, err)

	var got daemon.Frame
	require.NoError(t, json.Unmarshal(wire, &got))
	require.Equal(t, req.ID, got.ID)
	require.Equal(t, req.Method, got.Method)
	require.JSONEq(t, string(req.Params), string(got.Params))

	resp := daemon.Frame{ID: 7, Result: json.RawMessage(`{"subscription_id":42}`)}
	wire, err = json.Marshal(resp)
	require.NoError(t, err)
	var respGot daemon.Frame
	require.NoError(t, json.Unmarshal(wire, &respGot))
	require.Equal(t, uint64(7), respGot.ID)
	require.Empty(t, respGot.Method)
	require.JSONEq(t, `{"subscription_id":42}`, string(respGot.Result))
}

func TestFrame_ErrorOmitsResultAndViceVersa(t *testing.T) {
	errF := daemon.Frame{ID: 5, Error: &daemon.FrameError{
		Code: "not_found", ExitCode: 74, Message: "missing peer",
	}}
	wire, err := json.Marshal(errF)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(wire, &got))
	require.NotContains(t, got, "result")
	errMap := got["error"].(map[string]any)
	require.Equal(t, "not_found", errMap["code"])
	require.InDelta(t, 74, errMap["exit_code"], 0)
}

func TestMarshalUpdate_WrapsWatchEvent(t *testing.T) {
	ev := telegram.WatchEvent{
		Kind: telegram.EventNewMessage,
		Row:  output.MessageRow{ID: 9, Text: "hi"},
	}
	f, err := daemon.MarshalUpdate(42, ev)
	require.NoError(t, err)
	require.Equal(t, "update", f.Event)
	require.Equal(t, uint64(42), f.Sub)

	var roundTrip telegram.WatchEvent
	require.NoError(t, json.Unmarshal(f.Data, &roundTrip))
	require.Equal(t, telegram.EventNewMessage, roundTrip.Kind)
	require.Equal(t, 9, roundTrip.Row.ID)
}

func TestSubscribeParams_OmitemptyForEmptySlices(t *testing.T) {
	wire, err := json.Marshal(daemon.SubscribeParams{})
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(wire))

	wire, err = json.Marshal(daemon.SubscribeParams{
		Kinds: []string{"message"},
		Refs:  []string{"@chan"},
	})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(wire, &got))
	require.Contains(t, got, "kinds")
	require.Contains(t, got, "refs")
	require.NotContains(t, got, "peer_ids")
}
