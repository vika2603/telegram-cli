package config

import "errors"

// ErrInvalid marks config/env input that cannot be parsed or accepted.
var ErrInvalid = errors.New("invalid config")
