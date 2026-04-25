package output_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestRenderDownload(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	require.NoError(t, output.RenderDownload(ios, output.DownloadRow{
		MessageRef: "@chat:7",
		Path:       "photo.jpg",
		MIMEType:   "image/jpeg",
		Bytes:      12,
	}))

	got := stdout.String()
	require.Contains(t, got, "MESSAGE\tPATH\tTYPE\tBYTES")
	require.Contains(t, got, "@chat:7\tphoto.jpg\timage/jpeg\t12")
}
