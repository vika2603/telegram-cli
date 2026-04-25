package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/vika2603/telegram-cli/internal/program/status"
)

// EmitError writes err to stderr in the chosen format (human or json) and
// returns the exit code. Never touches stdout.
func EmitError(stderr io.Writer, mode string, err error) int {
	if err == nil {
		return 0
	}
	code := status.MapExitCode(err)
	if mode == "json" {
		obj := map[string]any{
			"error": map[string]any{
				"code":    status.Code(err),
				"message": err.Error(),
			},
			"exit_code": code,
		}
		b, _ := json.Marshal(obj) //nolint:errchkjson // obj holds only string/int values; Marshal cannot fail for these types
		_, _ = stderr.Write(append(b, '\n'))
	} else {
		_, _ = fmt.Fprintln(stderr, "error: "+err.Error())
	}
	return code
}
