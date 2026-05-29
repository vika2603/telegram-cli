package session

import (
	"errors"
	"fmt"

	"github.com/gotd/td/tgerr"
)

var (
	ErrAuth          = errors.New("auth required")
	ErrNetwork       = errors.New("network")
	ErrRateExhausted = errors.New("rate exhausted")
	ErrCurrent       = errors.New("refuses to terminate the current session")
	ErrBadPassword   = errors.New("2FA password incorrect")
)

type FloodWaitError struct {
	Seconds int
}

func (e *FloodWaitError) Error() string {
	return fmt.Sprintf("flood wait: %d seconds", e.Seconds)
}

func (e *FloodWaitError) Is(target error) bool {
	return target == ErrFloodWait
}

// ErrorDetail surfaces the retry-after seconds as a typed field on the
// JSON error envelope. Agents reading `error.retry_after_seconds`
// can compute a sleep instead of parsing the message string. The
// returned map keys are merged into the `error` object by
// output.EmitError; satisfies the (unimported) output.ErrorDetailer
// interface via structural typing.
func (e *FloodWaitError) ErrorDetail() map[string]any {
	return map[string]any{
		"retry_after_seconds": e.Seconds,
	}
}

var ErrFloodWait = errors.New("flood wait")

// AsFloodWait normalises the two shapes a flood-wait can take in this
// codebase into a single typed *FloodWaitError:
//
//  1. *FloodWaitError already wrapped on the error chain — produced
//     by ApplyFloodPolicy at some session boundary.
//  2. Raw *tgerr.Error of type FLOOD_WAIT — gotd's surface. Most
//     telegram-layer entry points do NOT route through
//     ApplyFloodPolicy today, so the raw form is what `status` and
//     `output` see in practice.
//
// Used by status.Code / status.MapExitCode / output.EmitError so the
// JSON envelope classifies and decorates a flood-wait the same way
// regardless of which call path produced it. Returns (nil, false)
// for anything else.
func AsFloodWait(err error) (*FloodWaitError, bool) {
	if err == nil {
		return nil, false
	}
	var typed *FloodWaitError
	if errors.As(err, &typed) {
		return typed, true
	}
	if d, ok := tgerr.AsFloodWait(err); ok {
		sec := int(d.Seconds())
		if sec == 0 {
			sec = 1
		}
		return &FloodWaitError{Seconds: sec}, true
	}
	return nil, false
}
