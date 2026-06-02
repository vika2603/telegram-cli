package status

// ErrorCode is the stable, machine-readable error code emitted as `error.code`
// in JSON output. It is tg's own enum — independent of the raw Telegram RPC
// enum, which (when present) is surfaced separately as `error.rpc_error`.
//
// This is the single source of truth for the code set; every value returned by
// Code(err) is one of these constants. The string values are part of the
// public output contract — do not change an existing one without updating the
// README and any downstream consumers.
type ErrorCode string

const (
	// CodeUnknown is the fallback for errors tg doesn't classify (exit 1).
	CodeUnknown ErrorCode = "unknown"

	CodeUsage         ErrorCode = "usage"             // command.ErrUsage / config.ErrInvalid (exit 2)
	CodeAuthRequired  ErrorCode = "auth_required"     // session.ErrAuth (exit 3)
	CodePeerNotFound  ErrorCode = "peer_not_found"    // peer.ErrNotFound (exit 4)
	CodePeerAmbiguous ErrorCode = "peer_ambiguous"    // peer.ErrAmbiguous (exit 4)
	CodeMessageNotFnd ErrorCode = "message_not_found" // message.ErrNotFound (exit 4)
	CodePeerForbidden ErrorCode = "peer_forbidden"    // peer.ErrForbidden / forbidden RPC (exit 5)
	CodeFloodWait     ErrorCode = "flood_wait"        // session.ErrFloodWait / raw FLOOD_WAIT (exit 6)
	CodeRateExhausted ErrorCode = "rate_exhausted"    // session.ErrRateExhausted (exit 6)
	CodeNetwork       ErrorCode = "network"           // session.ErrNetwork (exit 7)
	CodeCurrentSess   ErrorCode = "current_session"   // session.ErrCurrent (exit 8)
	CodeInvalidInvite ErrorCode = "invalid_invite"    // chat.ErrInvalidInvite (exit 8)
	CodeBadPassword   ErrorCode = "bad_password"      // session.ErrBadPassword (exit 8)
	CodeRevokeReq     ErrorCode = "revoke_required"   // message.ErrRevokeRequired (exit 8)
	CodePrecondition  ErrorCode = "precondition"      // command.ErrPrecondition (exit 9)
	CodeUnsupported   ErrorCode = "unsupported"       // command.ErrUnsupported (exit 9)
	CodeCacheMiss     ErrorCode = "cache_miss"        // peer.ErrCacheMiss (exit 9)
	CodeNoMedia       ErrorCode = "no_media"          // message.ErrNoMedia (exit 9)
	CodeNoLink        ErrorCode = "no_link"           // message.ErrNoLink (exit 9)
	CodeBusy          ErrorCode = "busy"              // account.ErrBusy (exit 72)
	CodeNotConfirmed  ErrorCode = "not_confirmed"     // command.ErrNotConfirmed (exit 73)
)
