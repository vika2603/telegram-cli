package telegram

import (
	"testing"
	"time"

	msgpeer "github.com/gotd/td/telegram/message/peer"
	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
)

func TestReverseMessageRows(t *testing.T) {
	rows := []output.MessageRow{{ID: 3}, {ID: 2}, {ID: 1}}
	reverseMessageRows(rows)
	require.Equal(t, []output.MessageRow{{ID: 1}, {ID: 2}, {ID: 3}}, rows)
}

func TestMessageToRow_UsesSenderEntity(t *testing.T) {
	entities := msgpeer.NewEntities(
		map[int64]*tg.User{
			7: {ID: 7, AccessHash: 70, FirstName: "Ada", LastName: "Lovelace", Username: "ada"},
		},
		nil,
		nil,
	)
	row := messageToRow(&tg.Message{
		ID:      10,
		Date:    int(time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC).Unix()),
		PeerID:  &tg.PeerUser{UserID: 7},
		FromID:  &tg.PeerUser{UserID: 7},
		Message: "hello",
	}, entities, "@ada")

	require.Equal(t, int64(7), row.FromID)
	require.Equal(t, "@ada:10", row.Ref)
	require.Equal(t, "user", row.FromKind)
	require.Equal(t, "Ada Lovelace", row.FromTitle)
	require.Equal(t, "ada", row.FromUsername)
	require.Equal(t, "@ada", row.FromRef)
}

func TestMessageToRow_FallsBackToPeerForChannelPost(t *testing.T) {
	entities := msgpeer.NewEntities(
		nil,
		nil,
		map[int64]*tg.Channel{
			42: {ID: 42, AccessHash: 420, Title: "News", Username: "news", Broadcast: true},
		},
	)
	row := messageToRow(&tg.Message{
		ID:      11,
		Date:    int(time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC).Unix()),
		PeerID:  &tg.PeerChannel{ChannelID: 42},
		Message: "post",
	}, entities, "@news")

	require.Equal(t, int64(-1_000_000_000_042), row.FromID)
	require.Equal(t, "@news:11", row.Ref)
	require.Equal(t, "channel", row.FromKind)
	require.Equal(t, "News", row.FromTitle)
	require.Equal(t, "news", row.FromUsername)
	require.Equal(t, "@news", row.FromRef)
}
