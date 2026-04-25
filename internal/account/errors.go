package account

import "errors"

// ErrBusy indicates another tg process currently owns this account's lock.
var ErrBusy = errors.New("account busy")
