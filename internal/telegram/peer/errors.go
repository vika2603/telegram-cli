package peer

import "errors"

var (
	ErrNotFound  = errors.New("peer not found")
	ErrForbidden = errors.New("peer access forbidden")
	ErrAmbiguous = errors.New("peer ambiguous")
	ErrCacheMiss = errors.New("cache miss")
)
