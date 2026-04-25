package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgmock"
	"github.com/stretchr/testify/require"

	actionsearch "github.com/vika2603/telegram-cli/internal/action/search"
)

func TestSearchMessages_GlobalUsersOnlyFiltersLocally(t *testing.T) {
	ctx := context.Background()
	mock := tgmock.NewRequire(t)
	api := tg.NewClient(mock)

	mock.ExpectCall(&tg.MessagesSearchGlobalRequest{
		Q:          "hello",
		Filter:     &tg.InputMessagesFilterEmpty{},
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      100,
	}).ThenResult(&tg.MessagesMessagesSlice{
		Count: 2,
		Messages: []tg.MessageClass{
			&tg.Message{
				ID:      2,
				PeerID:  &tg.PeerChannel{ChannelID: 42},
				Date:    int(time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC).Unix()),
				Message: "channel result",
			},
			&tg.Message{
				ID:      1,
				PeerID:  &tg.PeerUser{UserID: 7},
				FromID:  &tg.PeerUser{UserID: 7},
				Date:    int(time.Date(2026, 4, 24, 11, 0, 0, 0, time.UTC).Unix()),
				Message: "user result",
			},
		},
		Chats: []tg.ChatClass{
			&tg.Channel{ID: 42, AccessHash: 100, Title: "News", Broadcast: true, Photo: &tg.ChatPhotoEmpty{}},
		},
		Users: []tg.UserClass{
			&tg.User{ID: 7, AccessHash: 200, FirstName: "Ada", LastName: "Lovelace"},
		},
	})

	rows, err := SearchMessages(ctx, api, nil, actionsearch.MessageQuery{
		Query:     "hello",
		UsersOnly: true,
		Limit:     1,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 1, rows[0].MessageID)
	require.Equal(t, int64(7), rows[0].ChatID)
	require.Equal(t, "user", rows[0].ChatKind)
	require.Equal(t, "Ada Lovelace", rows[0].ChatTitle)
	require.Equal(t, int64(7), rows[0].FromID)
	require.Equal(t, "user result", rows[0].Text)
}
