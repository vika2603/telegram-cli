package account

import (
	"context"
	"time"
)

// SentCodeType enumerates the channels Telegram can use to deliver a login code.
// Unknown is a fallback for future gotd/Telegram types we haven't yet mapped.
type SentCodeType string

const (
	SentCodeApp        SentCodeType = "app"
	SentCodeSMS        SentCodeType = "sms"
	SentCodeCall       SentCodeType = "call"
	SentCodeFlashCall  SentCodeType = "flash_call"
	SentCodeMissedCall SentCodeType = "missed_call"
	SentCodeUnknown    SentCodeType = "unknown"
)

type SentCode struct {
	Type    SentCodeType
	Next    SentCodeType
	Timeout time.Duration
}

type TermsOfService struct {
	ID   string
	Text string
}

// UserAuthenticator drives code-based login (phone + SMS/app + 2FA).
// Implemented in cli/authprompt.go by ttyAuth / envAuth. The adapter in
// client/authadapter.go bridges this to gotd's auth.UserAuthenticator.
type UserAuthenticator interface {
	Phone(ctx context.Context) (string, error)
	Code(ctx context.Context, sent SentCode) (string, error)
	Password(ctx context.Context) (string, error)
	AcceptTOS(ctx context.Context, tos TermsOfService) error
}

// QRDisplay is the interface for QR-login UIs.
// Implemented in cli/authprompt.go by qrDisplay.
type QRDisplay interface {
	Show(ctx context.Context, url string) error
	Refresh(ctx context.Context, url string) error
	Done(ctx context.Context, accepted bool)
}
