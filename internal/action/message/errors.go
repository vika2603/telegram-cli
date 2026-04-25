package message

import "errors"

// ErrNoLink marks peers that cannot produce a t.me message link.
var ErrNoLink = errors.New("message link unavailable")
