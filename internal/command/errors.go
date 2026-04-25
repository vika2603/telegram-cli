package command

import (
	"errors"
	"fmt"
)

var (
	ErrUsage        = errors.New("usage")
	ErrPrecondition = errors.New("precondition failed")
	ErrUnsupported  = errors.New("operation not supported")
)

// FlagError wraps a flag-validation failure. Its Unwrap chain reaches
// ErrUsage, so program/status maps it to exit 64.
type FlagError struct{ err error }

func (e *FlagError) Error() string { return e.err.Error() }

// Unwrap exposes the inner error so errors.Is can walk the chain.
// ErrUsage is embedded via fmt.Errorf %w inside the inner error.
func (e *FlagError) Unwrap() error { return e.err }

// FlagErrorf builds a FlagError that wraps ErrUsage via fmt.Errorf.
func FlagErrorf(format string, a ...any) error {
	return &FlagError{err: fmt.Errorf("%w: "+format, append([]any{ErrUsage}, a...)...)}
}

// FlagErrorWrap promotes an arbitrary error into a usage-class error.
// errors.Is succeeds for both command.ErrUsage and the wrapped cause.
func FlagErrorWrap(err error) error {
	if err == nil {
		return nil
	}
	return &FlagError{err: fmt.Errorf("%w: %w", ErrUsage, err)}
}

// MutuallyExclusive returns a FlagError describing a multi-flag conflict
// when two or more conditions are true. When zero or one conditions are
// true the return is nil.
func MutuallyExclusive(message string, conditions ...bool) error {
	trueCount := 0
	for _, c := range conditions {
		if c {
			trueCount++
		}
	}
	if trueCount <= 1 {
		return nil
	}
	return FlagErrorf("%s", message)
}

// ErrSilent indicates the command already printed a human-readable
// explanation; the process must exit non-zero without printing anything
// further. Exit code 1.
var ErrSilent = errors.New("silent error")

// ErrCancel indicates the user canceled (Ctrl+C). Exit code 130.
var ErrCancel = errors.New("cancel")

// ErrNotConfirmed is returned when a destructive operation was declined at
// an interactive prompt.
var ErrNotConfirmed = errors.New("operation not confirmed")

// NoResultsError indicates an empty but successful result. Exit code 0;
// human mode prints the message, JSON mode emits an empty list/object.
type NoResultsError struct {
	message string
}

func (e *NoResultsError) Error() string { return e.message }

// Is reports whether target is any *NoResultsError, so MapExitCode can
// recognise the class without inspecting the message.
func (e *NoResultsError) Is(target error) bool {
	_, ok := target.(*NoResultsError)
	return ok
}

// NewNoResultsError constructs a NoResultsError with the given message.
func NewNoResultsError(message string) error { return &NoResultsError{message: message} }

// RequireFlag returns a usage error if value is empty.
func RequireFlag(name, value, msg string) error {
	if value == "" {
		return fmt.Errorf("%w: --%s is required: %s", ErrUsage, name, msg)
	}
	return nil
}
