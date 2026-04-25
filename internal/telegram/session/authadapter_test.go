package session

import (
	"context"
	"testing"
	"time"

	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/command"
)

type fakeAuth struct {
	phone      string
	code       string
	password   string
	acceptedID string
	lastSent   account.SentCode
}

func (f *fakeAuth) Phone(context.Context) (string, error) { return f.phone, nil }
func (f *fakeAuth) Code(_ context.Context, s account.SentCode) (string, error) {
	f.lastSent = s
	return f.code, nil
}
func (f *fakeAuth) Password(context.Context) (string, error) { return f.password, nil }
func (f *fakeAuth) AcceptTOS(_ context.Context, t account.TermsOfService) error {
	f.acceptedID = t.ID
	return nil
}

func TestAuthAdapter_SentCodeMapping_sms(t *testing.T) {
	fa := &fakeAuth{code: "12345"}
	ad := newAuthAdapter(fa, nil)
	sent := &tg.AuthSentCode{
		Type:    &tg.AuthSentCodeTypeSMS{},
		Timeout: 120,
	}
	_, err := ad.Code(context.Background(), sent)
	require.NoError(t, err)
	require.Equal(t, account.SentCodeSMS, fa.lastSent.Type)
	require.Equal(t, 120*time.Second, fa.lastSent.Timeout)
}

func TestAuthAdapter_SentCodeMapping_flashCall(t *testing.T) {
	fa := &fakeAuth{code: "x"}
	ad := newAuthAdapter(fa, nil)
	sent := &tg.AuthSentCode{Type: &tg.AuthSentCodeTypeFlashCall{}, Timeout: 0}
	_, err := ad.Code(context.Background(), sent)
	require.NoError(t, err)
	require.Equal(t, account.SentCodeFlashCall, fa.lastSent.Type)
}

func TestAuthAdapter_SignUp_returnsErrUnsupported(t *testing.T) {
	ad := newAuthAdapter(&fakeAuth{}, nil)
	_, err := ad.SignUp(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, command.ErrUnsupported)
}

func TestAuthAdapter_AcceptTOS_propagatesID(t *testing.T) {
	fa := &fakeAuth{}
	ad := newAuthAdapter(fa, nil)
	err := ad.AcceptTermsOfService(context.Background(), tg.HelpTermsOfService{
		ID:   tg.DataJSON{Data: `{"tos_id":"v42"}`},
		Text: "Terms",
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"tos_id":"v42"}`, fa.acceptedID)
}
