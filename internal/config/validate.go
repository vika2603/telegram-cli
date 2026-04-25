package config

import (
	"fmt"
)

// ValidateEnums checks merged config enum fields hold permitted values.
// Invalid values surface as usage errors (exit 64) before the logger is built.
func ValidateEnums(c Config) error {
	if c.Output.Format != nil {
		if err := oneOf("output.format", *c.Output.Format, "human", "json"); err != nil {
			return err
		}
	}
	if c.Output.Color != nil {
		if err := oneOf("output.color", *c.Output.Color, "auto", "always", "never"); err != nil {
			return err
		}
	}
	if c.Log.Level != nil {
		if err := oneOf("log.level", *c.Log.Level, "debug", "info", "warn", "error"); err != nil {
			return err
		}
	}
	if c.Log.Format != nil {
		if err := oneOf("log.format", *c.Log.Format, "console", "json"); err != nil {
			return err
		}
	}
	if c.FloodWait.Mode != nil {
		if err := oneOf("flood_wait.mode", *c.FloodWait.Mode, "fail", "wait"); err != nil {
			return err
		}
	}
	if c.FloodWait.MaxSeconds != nil && *c.FloodWait.MaxSeconds < 0 {
		return fmt.Errorf("%w: flood_wait.max_seconds must be >= 0, got %d", ErrInvalid, *c.FloodWait.MaxSeconds)
	}
	return nil
}

func oneOf(field, got string, allowed ...string) error {
	for _, a := range allowed {
		if got == a {
			return nil
		}
	}
	return fmt.Errorf("%w: %s=%q not in %v", ErrInvalid, field, got, allowed)
}
