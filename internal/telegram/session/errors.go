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

var ErrFloodWait = errors.New("flood wait")
