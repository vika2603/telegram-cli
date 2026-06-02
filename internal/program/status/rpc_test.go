package status

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gotd/td/tgerr"
	"github.com/stretchr/testify/require"
)

func rpcErr(typ string, code int) error {
	return &tgerr.Error{Code: code, Message: typ, Type: typ}
}

func TestRPC_ClassifiesForbiddenFamily(t *testing.T) {
	for _, typ := range []string{
		"CHAT_ADMIN_REQUIRED",
		"CHAT_WRITE_FORBIDDEN",
		"CHAT_SEND_DOCS_FORBIDDEN",
		"USER_PRIVACY_RESTRICTED",
		"USER_ADMIN_INVALID",
	} {
		err := rpcErr(typ, 403)
		require.Equal(t, "peer_forbidden", Code(err), typ)
		require.Equal(t, 5, MapExitCode(err), typ)
		require.NotContains(t, Message(err), "rpc error", typ)
		require.NotEmpty(t, Message(err), typ)
	}
}

func TestRPC_ClassifiesUsageFamily(t *testing.T) {
	for _, typ := range []string{"HIDE_REQUESTER_MISSING", "PARTICIPANT_ID_INVALID"} {
		err := rpcErr(typ, 400)
		require.Equal(t, "usage", Code(err), typ)
		require.Equal(t, 2, MapExitCode(err), typ)
		require.NotContains(t, Message(err), "rpc error", typ)
		require.NotEmpty(t, Message(err), typ)
	}
}

func TestRPC_MatchesThroughWrap(t *testing.T) {
	// Call sites often wrap the RPC error with context (e.g. "send media: %w").
	err := fmt.Errorf("send media: %w", rpcErr("CHAT_SEND_DOCS_FORBIDDEN", 403))
	require.Equal(t, "peer_forbidden", Code(err))
	require.Equal(t, 5, MapExitCode(err))
	require.Equal(t, "sending files is not allowed in this chat", Message(err))
}

func TestRPC_UnknownErrorUnchanged(t *testing.T) {
	err := errors.New("something else")
	require.Equal(t, "unknown", Code(err))
	require.Equal(t, 1, MapExitCode(err))
	require.Equal(t, "something else", Message(err))
}

func TestRPC_UnmappedRPCStaysUnknown(t *testing.T) {
	// An RPC error we don't classify must keep falling through to unknown.
	err := rpcErr("SOME_FUTURE_ERROR", 400)
	require.Equal(t, "unknown", Code(err))
	require.Equal(t, 1, MapExitCode(err))
}

func TestRPC_MessageNilEmpty(t *testing.T) {
	require.Empty(t, Message(nil))
}

func TestRPCType_SurfacesEnum(t *testing.T) {
	require.Equal(t, "CHAT_ADMIN_REQUIRED", RPCType(rpcErr("CHAT_ADMIN_REQUIRED", 403)))
	require.Equal(t, "SOME_FUTURE_ERROR", RPCType(rpcErr("SOME_FUTURE_ERROR", 400)))
	require.Equal(t, "CHAT_SEND_DOCS_FORBIDDEN",
		RPCType(fmt.Errorf("send media: %w", rpcErr("CHAT_SEND_DOCS_FORBIDDEN", 403))))
	require.Empty(t, RPCType(errors.New("not an rpc error")))
	require.Empty(t, RPCType(nil))
}
