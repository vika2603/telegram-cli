package status

import (
	"errors"

	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
)

// rpcClass is the classification for a known raw Telegram RPC error type: the
// stable JSON code, the process exit code, and a friendly user-facing message.
type rpcClass struct {
	code    ErrorCode
	exit    int
	message string
}

// rpcErrors maps known raw Telegram RPC error types to a classification. These
// are errors that the telegram layer returns unwrapped and would otherwise
// reach the user as "unknown" with an opaque "rpc error code ...: FOO" string.
// Keys reference gotd's generated tg.Err* constants so they can't drift from
// the wire enum.
//
// Errors that are already converted near their call site (e.g.
// CHAT_PUBLIC_REQUIRED -> ErrUnsupported, INVITE_HASH_EXPIRED -> ErrInvalidInvite,
// USER_ALREADY_PARTICIPANT -> success) are intentionally absent here so their
// existing handling wins.
var rpcErrors = map[string]rpcClass{
	// "Action forbidden" family -> peer_forbidden / exit 5.
	tg.ErrChatAdminRequired:         {CodePeerForbidden, 5, "this action requires admin rights in the chat"},
	tg.ErrChatWriteForbidden:        {CodePeerForbidden, 5, "posting is not allowed in this chat"},
	tg.ErrChatSendPlainForbidden:    {CodePeerForbidden, 5, "sending text messages is not allowed in this chat"},
	tg.ErrChatSendMediaForbidden:    {CodePeerForbidden, 5, "sending media is not allowed in this chat"},
	tg.ErrChatSendDocsForbidden:     {CodePeerForbidden, 5, "sending files is not allowed in this chat"},
	tg.ErrChatSendPhotosForbidden:   {CodePeerForbidden, 5, "sending photos is not allowed in this chat"},
	tg.ErrChatSendVideosForbidden:   {CodePeerForbidden, 5, "sending videos is not allowed in this chat"},
	tg.ErrChatSendStickersForbidden: {CodePeerForbidden, 5, "sending stickers is not allowed in this chat"},
	tg.ErrChatSendGifsForbidden:     {CodePeerForbidden, 5, "sending GIFs is not allowed in this chat"},
	tg.ErrChatSendPollForbidden:     {CodePeerForbidden, 5, "sending polls is not allowed in this chat"},
	tg.ErrUserPrivacyRestricted:     {CodePeerForbidden, 5, "the user's privacy settings don't allow this"},
	tg.ErrUserAdminInvalid:          {CodePeerForbidden, 5, "you can't edit this admin's rights"},

	// "Bad request argument" family -> usage / exit 2.
	tg.ErrHideRequesterMissing: {CodeUsage, 2, "no pending join request from this user"},
	tg.ErrParticipantIDInvalid: {CodeUsage, 2, "that user can't be the target of this action (e.g. the group creator)"},
	tg.ErrUserCreator:          {CodeUsage, 2, "this action isn't allowed on the group/channel creator"},
	tg.ErrFileReferenceExpired: {CodeUsage, 2, "sticker reference expired; run `tg msg sticker list` again for a fresh ref"},
	tg.ErrFileReferenceInvalid: {CodeUsage, 2, "sticker reference is no longer valid; run `tg msg sticker list` again for a fresh ref"},
	tg.ErrStickersetInvalid:    {CodeUsage, 2, "no such sticker set (check the short name or addstickers link)"},
}

// matchRPC returns the classification for a known raw Telegram RPC error
// anywhere on err's chain, if any.
func matchRPC(err error) (rpcClass, bool) {
	for typ, cls := range rpcErrors {
		if tgerr.Is(err, typ) {
			return cls, true
		}
	}
	return rpcClass{}, false
}

// RPCType returns the raw Telegram RPC error enum (e.g. "CHAT_ADMIN_REQUIRED")
// from anywhere on err's chain, or "" if err is not a Telegram RPC error. It is
// surfaced alongside the friendly message so callers keep the exact enum for
// programmatic matching, even for errors we don't friendly-map.
func RPCType(err error) string {
	var rpcErr *tgerr.Error
	if errors.As(err, &rpcErr) {
		return rpcErr.Type
	}
	return ""
}

// Message returns a user-facing message for err. For a known raw Telegram RPC
// error that would otherwise surface as an opaque "rpc error code ...: FOO"
// string, it returns a friendly explanation; otherwise it returns err.Error().
func Message(err error) string {
	if err == nil {
		return ""
	}
	// A sentinel-classified error already carries an authored message (the
	// fmt.Errorf wrap); only override the message for raw RPC errors that the
	// sentinel path doesn't handle. This keeps Message in lockstep with Code.
	if _, ok := sentinelCode(err); ok {
		return err.Error()
	}
	if cls, ok := matchRPC(err); ok {
		return cls.message
	}
	return err.Error()
}
