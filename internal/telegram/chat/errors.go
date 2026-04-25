package chat

import "errors"

// ErrInvalidInvite marks expired, revoked, or malformed invite links.
var ErrInvalidInvite = errors.New("invite link invalid or expired")
