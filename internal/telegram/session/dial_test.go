package session

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeviceConfig_MapsSessionOptionsToGotdDevice(t *testing.T) {
	got := deviceConfig(DeviceOptions{
		Model:          "tg CLI",
		SystemVersion:  "darwin/arm64",
		AppVersion:     "1.2.3",
		SystemLangCode: "en",
		LangCode:       "en",
	})

	require.Equal(t, "tg CLI", got.DeviceModel)
	require.Equal(t, "darwin/arm64", got.SystemVersion)
	require.Equal(t, "1.2.3", got.AppVersion)
	require.Equal(t, "en", got.SystemLangCode)
	require.Equal(t, "en", got.LangCode)
}
