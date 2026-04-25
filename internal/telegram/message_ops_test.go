package telegram

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
)

func TestSentMessagesDedupesUpdateMessageID(t *testing.T) {
	rows := sentMessages(&tg.Updates{Updates: []tg.UpdateClass{
		&tg.UpdateMessageID{ID: 9},
		&tg.UpdateNewMessage{Message: &tg.Message{
			ID:     9,
			Date:   1,
			PeerID: &tg.PeerUser{UserID: 100},
		}},
	}})

	require.Equal(t, []output.SendResultRow{{
		MessageID: 9,
		ChatID:    100,
		Date:      "1970-01-01T00:00:01Z",
	}}, rows)
}

func TestMediaOutputPath(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(wd))
	})

	got, err := mediaOutputPath("", "../photo.jpg")
	require.NoError(t, err)
	require.Equal(t, "photo.jpg", got)

	got, err = mediaOutputPath(dir, "photo.jpg")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "photo.jpg"), got)

	got, err = mediaOutputPath(filepath.Join(dir, "custom.bin"), "photo.jpg")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "custom.bin"), got)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("old"), 0o600))
	got, err = mediaOutputPath("", "photo.jpg")
	require.NoError(t, err)
	require.Equal(t, "photo-1.jpg", got)
}
