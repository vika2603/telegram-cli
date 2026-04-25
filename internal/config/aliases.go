package config

// Aliases returns the aliases map from c, or an empty map if unset.
// Helper so callers don't have to nil-check in the hot path.
func (c Config) AliasesOrEmpty() map[string]string {
	if c.Aliases == nil {
		return map[string]string{}
	}
	return c.Aliases
}
