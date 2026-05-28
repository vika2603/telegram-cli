package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/vika2603/telegram-cli/internal/program/status"
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
	if mode == "json" {
		errObj := map[string]any{
			"code":    status.Code(err),
			"message": err.Error(),
		}
		var d ErrorDetailer
		if errors.As(err, &d) {
			for k, v := range d.ErrorDetail() {
				// Don't let detailers overwrite the base shape — code
				// and message stay authoritative.
				if k == "code" || k == "message" {
					continue
				}
				errObj[k] = v
			}
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
