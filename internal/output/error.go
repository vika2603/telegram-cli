package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/program/status"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
)

// ErrorDetailer is implemented by error types that want to surface
// extra structured fields under the `error` object in JSON output.
// Typical use: a flood-wait error that knows the retry-after seconds
// returns {"retry_after_seconds": N} so agents can read the wait time
// from a typed field instead of parsing the error message string.
//
// The interface is intentionally minimal so error types in other
// packages (notably internal/telegram/session) can satisfy it via Go's
// structural typing without importing internal/output. EmitError uses
// errors.As to find an implementing error anywhere on the wrap chain.
type ErrorDetailer interface {
	ErrorDetail() map[string]any
}

// EmitError writes err to stderr in the chosen format (human or json) and
// returns the exit code. Never touches stdout.
func EmitError(stderr io.Writer, mode string, err error) int {
	if err == nil {
		return 0
	}
	code := status.MapExitCode(err)
	// A user cancellation (Ctrl+C / SIGINT) is not worth shouting about in a
	// terminal — the shell already echoed "^C". Exit quietly with 130.
	// JSON mode still emits the structured record so scripts can detect it.
	if mode != "json" && errors.Is(err, command.ErrCancel) {
		return code
	}
	if mode == "json" {
		errObj := map[string]any{
			"code":    status.Code(err),
			"message": err.Error(),
		}
		mergeDetail := func(detail map[string]any) {
			for k, v := range detail {
				// Don't let detailers overwrite the base shape — code
				// and message stay authoritative.
				if k == "code" || k == "message" {
					continue
				}
				errObj[k] = v
			}
		}
		var d ErrorDetailer
		if errors.As(err, &d) {
			mergeDetail(d.ErrorDetail())
		} else if fwe, ok := session.AsFloodWait(err); ok {
			// Fallback for raw gotd FLOOD_WAIT errors that never went
			// through ApplyFloodPolicy and so carry no typed
			// ErrorDetailer on the chain. AsFloodWait synthesises a
			// FloodWaitError so retry_after_seconds still surfaces.
			mergeDetail(fwe.ErrorDetail())
		}
		obj := map[string]any{
			"error":     errObj,
			"exit_code": code,
		}
		b, _ := json.Marshal(obj) //nolint:errchkjson // obj holds only string/int/map values; Marshal cannot fail for these types
		_, _ = stderr.Write(append(b, '\n'))
	} else {
		_, _ = fmt.Fprintln(stderr, "error: "+err.Error())
	}
	return code
}
