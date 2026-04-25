package output

// ConfigKeyRow is emitted by config get / set / unset. Source is only
// populated by get (it indicates the precedence layer the value came from).
// Old / New are populated by set / unset. Value is populated by get.
//
// Value / Old / New are any so typed values (int64, bool, string) survive the
// JSON round-trip correctly — api_id stays numeric rather than stringified.
// The typed value is produced by config.CoerceValue (set) or config.ReadRaw
// (get / unset old).
type ConfigKeyRow struct {
	Key    string `json:"key"`
	Value  any    `json:"value,omitempty"`
	Old    any    `json:"old,omitempty"`
	New    any    `json:"new,omitempty"`
	Source string `json:"source,omitempty"`
	Action string `json:"action,omitempty"`
}
