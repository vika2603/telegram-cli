package runtime_test

import (
	goruntime "runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/config"
	appruntime "github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
)

func ptr[T any](v T) *T { return &v }

func cfgInvocation(mode string, maxSec int) *appruntime.Invocation {
	return &appruntime.Invocation{
		Config: func() (*config.Config, error) {
			return &config.Config{
				FloodWait: config.FloodWaitCfg{Mode: ptr(mode), MaxSeconds: ptr(maxSec)},
			}, nil
		},
	}
}

// TestClientOptsFrom_FloodDefaults: no config, no flag → fail mode, 30s.
func TestClientOptsFrom_FloodDefaults(t *testing.T) {
	opts := appruntime.ClientOptsFrom(&appruntime.Invocation{}, &account.Account{})
	require.Equal(t, session.FloodFail, opts.FloodMode)
	require.Equal(t, 30, opts.FloodMaxSec)
}

// TestClientOptsFrom_FloodFromConfig: config wait mode + custom cap is honored.
func TestClientOptsFrom_FloodFromConfig(t *testing.T) {
	opts := appruntime.ClientOptsFrom(cfgInvocation("wait", 90), &account.Account{})
	require.Equal(t, session.FloodWait, opts.FloodMode)
	require.Equal(t, 90, opts.FloodMaxSec)
}

// TestClientOptsFrom_FloodFlagOverridesConfigCap: --flood-wait-max wins
// over the config max_seconds, but mode still comes from config.
func TestClientOptsFrom_FloodFlagOverridesConfigCap(t *testing.T) {
	f := cfgInvocation("wait", 90)
	f.FloodWaitMax = ptr(5)
	opts := appruntime.ClientOptsFrom(f, &account.Account{})
	require.Equal(t, session.FloodWait, opts.FloodMode, "mode still from config")
	require.Equal(t, 5, opts.FloodMaxSec, "cap overridden by flag")
}

// TestClientOptsFrom_WaitFlagOverridesConfigMode: --wait forces wait mode
// even when config says fail; --no-wait forces fail even when config says wait.
func TestClientOptsFrom_WaitFlagOverridesConfigMode(t *testing.T) {
	wait := cfgInvocation("fail", 30)
	wait.WaitFlood = ptr(true)
	require.Equal(t, session.FloodWait, appruntime.ClientOptsFrom(wait, &account.Account{}).FloodMode,
		"--wait must override config fail mode")

	noWait := cfgInvocation("wait", 30)
	noWait.WaitFlood = ptr(false)
	require.Equal(t, session.FloodFail, appruntime.ClientOptsFrom(noWait, &account.Account{}).FloodMode,
		"--no-wait must override config wait mode")
}

// TestClientOptsFrom_FloodConfigErrorFallsBackToDefaults: a config
// closure that errors must not crash — defaults apply.
func TestClientOptsFrom_FloodConfigErrorFallsBackToDefaults(t *testing.T) {
	f := &appruntime.Invocation{
		Config: func() (*config.Config, error) { return nil, configBoomError{} },
	}
	opts := appruntime.ClientOptsFrom(f, &account.Account{})
	require.Equal(t, session.FloodFail, opts.FloodMode)
	require.Equal(t, 30, opts.FloodMaxSec)
}

type configBoomError struct{}

func (configBoomError) Error() string { return "config boom" }

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
