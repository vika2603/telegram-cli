package telegram

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gotd/td/crypto"
	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
)

// TestRandomIDReader locks the seeding contract SendMessage/ForwardMessages
// rely on, reading ids exactly as gotd does (crypto.RandInt64):
//   - the FIRST id equals the seed, so a single-message send/forward puts
//     --random-id on the wire verbatim;
//   - later ids are distinct, so albums and multi-message forwards get one
//     unique id each;
//   - the same seed reproduces the same sequence, so a retry dedupes.
func TestRandomIDReader(t *testing.T) {
	for _, seed := range []int64{1, 42, 7654321, -1, 9223372036854775807, -9223372036854775808} {
		r := randomIDReader(seed)
		first, err := crypto.RandInt64(r)
		require.NoError(t, err)
		require.Equal(t, seed, first, "first id must equal the seed")

		second, err := crypto.RandInt64(r)
		require.NoError(t, err)
		require.NotEqual(t, first, second, "subsequent ids must differ")

		third, err := crypto.RandInt64(r)
		require.NoError(t, err)
		require.NotEqual(t, second, third)

		// Same seed reproduces the same sequence.
		r2 := randomIDReader(seed)
		a, _ := crypto.RandInt64(r2)
		b, _ := crypto.RandInt64(r2)
		c, _ := crypto.RandInt64(r2)
		require.Equal(t, []int64{first, second, third}, []int64{a, b, c})
	}
}

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
