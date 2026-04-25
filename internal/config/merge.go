package config

// Merge overlays over on top of base: each non-nil field in over wins.
// Aliases maps are replaced wholesale (not unioned) — simpler mental model.
func Merge(base, over Config) Config {
	if over.Version != nil {
		base.Version = over.Version
	}
	if over.DefaultAccount != nil {
		base.DefaultAccount = over.DefaultAccount
	}
	if over.APIID != nil {
		base.APIID = over.APIID
	}
	if over.APIHash != nil {
		base.APIHash = over.APIHash
	}
	if over.Output.Format != nil {
		base.Output.Format = over.Output.Format
	}
	if over.Output.Color != nil {
		base.Output.Color = over.Output.Color
	}
	if over.Log.Level != nil {
		base.Log.Level = over.Log.Level
	}
	if over.Log.File != nil {
		base.Log.File = over.Log.File
	}
	if over.Log.Format != nil {
		base.Log.Format = over.Log.Format
	}
	if over.FloodWait.Mode != nil {
		base.FloodWait.Mode = over.FloodWait.Mode
	}
	if over.FloodWait.MaxSeconds != nil {
		base.FloodWait.MaxSeconds = over.FloodWait.MaxSeconds
	}
	if over.Aliases != nil {
		base.Aliases = over.Aliases
	}
	return base
}
