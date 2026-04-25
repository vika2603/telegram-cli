package message

import "errors"

var (
	ErrNotFound       = errors.New("message not found")
	ErrNoMedia        = errors.New("no media")
	ErrNoLink         = errors.New("no link")
	ErrRevokeRequired = errors.New("message delete requires --revoke")
)
