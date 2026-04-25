package ref

// ExpandAlias substitutes s with its alias value if the map has an entry,
// otherwise returns s unchanged. Single-level only: the result is NOT re-expanded.
func ExpandAlias(s string, aliases map[string]string) string {
	if v, ok := aliases[s]; ok {
		return v
	}
	return s
}
