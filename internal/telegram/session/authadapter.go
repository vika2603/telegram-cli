package session

import (
	"context"
	"fmt"
	"time"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/command"
)

// authAdapter implements gotd's auth.UserAuthenticator by delegating each
// callback to an account.UserAuthenticator. This is the only place the two
// interfaces touch each other.
type authAdapter struct {
	inner  account.UserAuthenticator
	logger *zap.Logger
}

var _ auth.UserAuthenticator = (*authAdapter)(nil)

func newAuthAdapter(inner account.UserAuthenticator, logger *zap.Logger) *authAdapter {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &authAdapter{inner: inner, logger: logger}
}

func (a *authAdapter) Phone(ctx context.Context) (string, error) {
	return a.inner.Phone(ctx)
}

func (a *authAdapter) Code(ctx context.Context, sent *tg.AuthSentCode) (string, error) {
	s := account.SentCode{
		Type:    mapSentCodeType(sent.Type, a.logger),
		Next:    mapNextType(sent.NextType),
		Timeout: time.Duration(sent.Timeout) * time.Second,
	}
	return a.inner.Code(ctx, s)
}

func (a *authAdapter) Password(ctx context.Context) (string, error) {
	return a.inner.Password(ctx)
}

func (a *authAdapter) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	return a.inner.AcceptTOS(ctx, account.TermsOfService{
		ID:   tos.ID.Data,
		Text: tos.Text,
	})
}

func (a *authAdapter) SignUp(_ context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("SignUp: %w", command.ErrUnsupported)
}

func mapSentCodeType(t tg.AuthSentCodeTypeClass, log *zap.Logger) account.SentCodeType {
	switch t.(type) {
	case *tg.AuthSentCodeTypeApp:
		return account.SentCodeApp
	case *tg.AuthSentCodeTypeSMS:
		return account.SentCodeSMS
	case *tg.AuthSentCodeTypeCall:
		return account.SentCodeCall
	case *tg.AuthSentCodeTypeFlashCall:
		return account.SentCodeFlashCall
	case *tg.AuthSentCodeTypeMissedCall:
		return account.SentCodeMissedCall
	default:
		log.Warn("unknown SentCode variant", zap.String("type", fmt.Sprintf("%T", t)))
		return account.SentCodeUnknown
	}
}

// mapNextType maps the resend-channel type. gotd uses a different interface
// (AuthCodeTypeClass) for the next-type field — unknown variants map to
// SentCodeUnknown.
func mapNextType(t tg.AuthCodeTypeClass) account.SentCodeType {
	if t == nil {
		return ""
	}
	switch t.(type) {
	case *tg.AuthCodeTypeSMS:
		return account.SentCodeSMS
	case *tg.AuthCodeTypeCall:
		return account.SentCodeCall
	case *tg.AuthCodeTypeFlashCall:
		return account.SentCodeFlashCall
	case *tg.AuthCodeTypeMissedCall:
		return account.SentCodeMissedCall
	default:
		return account.SentCodeUnknown
	}
}
