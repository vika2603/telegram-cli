package telegram

import (
	"testing"

	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
)

func TestSearchPeerToChatRow_User(t *testing.T) {
	row := searchPeerToChatRow(&tg.PeerUser{UserID: 10}, &tg.ContactsFound{
		Users: []tg.UserClass{&tg.User{ID: 10, FirstName: "Pavel", LastName: "Durov", Username: "durov"}},
	})

	require.Equal(t, output.ChatRow{
		ID:       10,
		Ref:      "@durov",
		Kind:     "user",
		Title:    "Pavel Durov",
		Username: "durov",
	}, row)
}

func TestSearchPeerToChatRow_Bot(t *testing.T) {
	row := searchPeerToChatRow(&tg.PeerUser{UserID: 11}, &tg.ContactsFound{
		Users: []tg.UserClass{&tg.User{ID: 11, FirstName: "Build", Username: "buildbot", Bot: true}},
	})

	require.Equal(t, "bot", row.Kind)
	require.Equal(t, "Build", row.Title)
}

func TestSearchPeerToChatRow_Chat(t *testing.T) {
	row := searchPeerToChatRow(&tg.PeerChat{ChatID: 20}, &tg.ContactsFound{
		Chats: []tg.ChatClass{&tg.Chat{ID: 20, Title: "Team"}},
	})

	require.Equal(t, output.ChatRow{ID: -20, Ref: "g:20", Kind: "chat", Title: "Team"}, row)
}

func TestSearchPeerToChatRow_Channel(t *testing.T) {
	row := searchPeerToChatRow(&tg.PeerChannel{ChannelID: 30}, &tg.ContactsFound{
		Chats: []tg.ChatClass{&tg.Channel{ID: 30, AccessHash: 300, Title: "News", Username: "news", Broadcast: true}},
	})

	require.Equal(t, output.ChatRow{
		ID:       -1_000_000_000_030,
		Ref:      "@news",
		Kind:     "channel",
		Title:    "News",
		Username: "news",
	}, row)
}

func TestSearchPeerToChatRow_Supergroup(t *testing.T) {
	row := searchPeerToChatRow(&tg.PeerChannel{ChannelID: 31}, &tg.ContactsFound{
		Chats: []tg.ChatClass{&tg.Channel{ID: 31, Title: "Group", Broadcast: false}},
	})

	require.Equal(t, "chat", row.Kind)
	require.Equal(t, int64(-1_000_000_000_031), row.ID)
}
