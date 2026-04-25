package runtime_test

import (
	goruntime "runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	appruntime "github.com/vika2603/telegram-cli/internal/runtime"
)

func TestClientOptsFrom_SetsTelegramDeviceIdentity(t *testing.T) {
	opts := appruntime.ClientOptsFrom(&appruntime.Invocation{AppVersion: "1.2.3"}, &account.Account{
		Meta: account.Meta{APIID: 123, APIHash: "hash"},
	})

	require.Equal(t, 123, opts.APIID)
	require.Equal(t, "hash", opts.APIHash)
	require.Equal(t, "tg CLI", opts.Device.Model)
	require.Equal(t, goruntime.GOOS+"/"+goruntime.GOARCH, opts.Device.SystemVersion)
	require.Equal(t, "1.2.3", opts.Device.AppVersion)
	require.Equal(t, "en", opts.Device.SystemLangCode)
	require.Equal(t, "en", opts.Device.LangCode)
}

func TestClientOptsFrom_DeviceAppVersionFallback(t *testing.T) {
	opts := appruntime.ClientOptsFrom(nil, &account.Account{})

	require.Equal(t, "dev", opts.Device.AppVersion)
}
