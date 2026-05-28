package session

import (
	"errors"
	"fmt"
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
