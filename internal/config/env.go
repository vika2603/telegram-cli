package config

import (
	"fmt"
	"os"
	"strconv"
)

// FromEnv builds a sparse Config from TG_* environment variables.
// Only variables that are actually set contribute; unset variables leave
// the field nil so Merge does not overwrite lower tiers.
//
// Malformed integer vars (TG_API_ID, TG_FLOOD_WAIT_MAX) surface as
// ErrInvalid — silently dropping them would let the caller fall back
// to defaults and hide the user's intent. Enum string vars are not
// validated here so misspellings surface after the full merge with the
// right line of provenance.
func FromEnv() (Config, error) {
	var c Config
	if v, ok := os.LookupEnv("TG_ACCOUNT"); ok && v != "" {
		c.DefaultAccount = &v
	}
	if v, ok := os.LookupEnv("TG_API_ID"); ok && v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("%w: TG_API_ID=%q is not a valid integer", ErrInvalid, v)
		}
		c.APIID = &n
	}
	if v, ok := os.LookupEnv("TG_API_HASH"); ok && v != "" {
		c.APIHash = &v
	}
	if v, ok := os.LookupEnv("TG_OUTPUT"); ok && v != "" {
		c.Output.Format = &v
	}
	if v, ok := os.LookupEnv("TG_COLOR"); ok && v != "" {
		c.Output.Color = &v
	}
	if v, ok := os.LookupEnv("TG_LOG_LEVEL"); ok && v != "" {
		c.Log.Level = &v
	}
	// TG_LOG_FILE="" is meaningful: "reset to stderr". Unlike other enum-
	// style vars, the empty value is a valid explicit choice, so we honor it.
	if v, ok := os.LookupEnv("TG_LOG_FILE"); ok {
		c.Log.File = &v
	}
	if v, ok := os.LookupEnv("TG_LOG_FORMAT"); ok && v != "" {
		c.Log.Format = &v
	}
	if v, ok := os.LookupEnv("TG_FLOOD_WAIT_MODE"); ok && v != "" {
		c.FloodWait.Mode = &v
	}
	if v, ok := os.LookupEnv("TG_FLOOD_WAIT_MAX"); ok && v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("%w: TG_FLOOD_WAIT_MAX=%q is not a valid integer", ErrInvalid, v)
		}
		if n < 0 {
			return Config{}, fmt.Errorf("%w: TG_FLOOD_WAIT_MAX=%d must be >= 0", ErrInvalid, n)
		}
		c.FloodWait.MaxSeconds = &n
	}
	return c, nil
}
