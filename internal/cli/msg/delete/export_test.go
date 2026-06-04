package deletecmd

import "encoding/json"

// DecodeDeleteCountForTest exposes decodeDeleteCount to the external test
// package.
func DecodeDeleteCountForTest(raw json.RawMessage, requested int) int {
	return decodeDeleteCount(raw, requested)
}
