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

func TestExtractEntities_TextURLAndBasicTypes(t *testing.T) {
	text := "Pyrois rules"
	ents := []tg.MessageEntityClass{
		&tg.MessageEntityTextURL{Offset: 0, Length: 6, URL: "https://example.com/p"},
		&tg.MessageEntityBold{Offset: 7, Length: 5},
		&tg.MessageEntityURL{Offset: 0, Length: 6},
	}
	got := extractEntities(text, ents)
	require.Len(t, got, 3)
	require.Equal(t, output.MessageEntity{Type: "text_url", Text: "Pyrois", URL: "https://example.com/p"}, got[0])
	require.Equal(t, output.MessageEntity{Type: "bold", Text: "rules"}, got[1])
	require.Equal(t, output.MessageEntity{Type: "url", Text: "Pyrois", URL: "Pyrois"}, got[2])
}

func TestExtractEntities_UTF16Offsets(t *testing.T) {
	// 🚀 is a surrogate pair in UTF-16 (2 code units), so an entity at offset 2
	// should land on the rune that follows it.
	text := "🚀ok"
	ents := []tg.MessageEntityClass{
		&tg.MessageEntityBold{Offset: 2, Length: 2},
	}
	got := extractEntities(text, ents)
	require.Len(t, got, 1)
	require.Equal(t, "ok", got[0].Text)
}

func TestExtractEntities_OutOfRangeIsIgnored(t *testing.T) {
	got := extractEntities("hi", []tg.MessageEntityClass{
		&tg.MessageEntityBold{Offset: 10, Length: 5},
	})
	require.Len(t, got, 1)
	require.Empty(t, got[0].Text)
	require.Equal(t, "bold", got[0].Type)
}

func TestExtractForward_ChannelPostBuildsLink(t *testing.T) {
	entities := msgpeer.NewEntities(
		nil,
		nil,
		map[int64]*tg.Channel{
			99: {ID: 99, AccessHash: 990, Title: "nanoka.cc News", Username: "nanoka_news", Broadcast: true},
		},
	)
	fwd := tg.MessageFwdHeader{Date: int(time.Date(2026, 5, 23, 2, 13, 21, 0, time.UTC).Unix())}
	fwd.SetFromID(&tg.PeerChannel{ChannelID: 99})
	fwd.SetChannelPost(153)
	fwd.SetPostAuthor("ed")

	got := extractForward(fwd, entities)
	require.NotNil(t, got)
	require.Equal(t, "2026-05-23T02:13:21Z", got.Date)
	require.Equal(t, 153, got.ChannelPostID)
	require.Equal(t, "ed", got.PostAuthor)
	require.Equal(t, "https://t.me/nanoka_news/153", got.Link)
	require.NotNil(t, got.From)
	require.Equal(t, "channel", got.From.Type)
	require.Equal(t, "nanoka_news", got.From.Username)
	require.Equal(t, "https://t.me/nanoka_news", got.From.Link)
	require.Equal(t, "nanoka.cc News", got.From.Title)
}

func TestExtractForward_HiddenSenderFromName(t *testing.T) {
	fwd := tg.MessageFwdHeader{Date: int(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix())}
	fwd.SetFromName("Anonymous")

	got := extractForward(fwd, msgpeer.NewEntities(nil, nil, nil))
	require.NotNil(t, got)
	require.Nil(t, got.From)
	require.Equal(t, "Anonymous", got.FromName)
	require.Empty(t, got.Link)
}

func TestExtractForward_EmptyReturnsNil(t *testing.T) {
	got := extractForward(tg.MessageFwdHeader{}, msgpeer.NewEntities(nil, nil, nil))
	require.Nil(t, got)
}

func TestExtractButtons_InlineURLAndCallback(t *testing.T) {
	rm := &tg.ReplyInlineMarkup{Rows: []tg.KeyboardButtonRow{
		{Buttons: []tg.KeyboardButtonClass{
			&tg.KeyboardButtonURL{Text: "Open", URL: "https://e.com"},
			&tg.KeyboardButtonCallback{Text: "Tap"},
		}},
		{Buttons: []tg.KeyboardButtonClass{
			&tg.KeyboardButtonURLAuth{Text: "Auth", URL: "https://e.com/auth"},
		}},
	}}
	got := extractButtons(rm)
	require.Len(t, got, 3)
	require.Equal(t, output.MessageButton{Row: 0, Text: "Open", URL: "https://e.com", Type: "url"}, got[0])
	require.Equal(t, output.MessageButton{Row: 0, Text: "Tap", Type: "callback"}, got[1])
	require.Equal(t, output.MessageButton{Row: 1, Text: "Auth", URL: "https://e.com/auth", Type: "url_auth"}, got[2])
}

func TestExtractButtons_NonInlineIsIgnored(t *testing.T) {
	require.Nil(t, extractButtons(&tg.ReplyKeyboardMarkup{}))
}

func TestMessageToRow_PopulatesForwardAndEntities(t *testing.T) {
	entities := msgpeer.NewEntities(
		nil,
		nil,
		map[int64]*tg.Channel{
			99: {ID: 99, AccessHash: 990, Title: "Src", Username: "src", Broadcast: true},
		},
	)
	m := &tg.Message{
		ID:      77,
		Date:    int(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).Unix()),
		PeerID:  &tg.PeerChannel{ChannelID: 99},
		Message: "click me",
	}
	m.SetEntities([]tg.MessageEntityClass{
		&tg.MessageEntityTextURL{Offset: 0, Length: 8, URL: "https://e.com"},
	})
	fwd := tg.MessageFwdHeader{Date: int(time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC).Unix())}
	fwd.SetFromID(&tg.PeerChannel{ChannelID: 99})
	fwd.SetChannelPost(11)
	m.SetFwdFrom(fwd)

	row := messageToRow(m, entities, "@dst")
	require.Len(t, row.Entities, 1)
	require.Equal(t, "https://e.com", row.Entities[0].URL)
	require.NotNil(t, row.Forward)
	require.Equal(t, 11, row.Forward.ChannelPostID)
	require.Equal(t, "https://t.me/src/11", row.Forward.Link)
}

// TestMessageToRow_PopulatesEditDateGroupedReactions exercises the
// three agent-CLI fields layered on top of PR #2: EditDate, GroupedID,
// and a Reactions list with one self-reacted emoji and one custom
// emoji to cover both kinds the gotd schema knows about.
func TestMessageToRow_PopulatesEditDateGroupedReactions(t *testing.T) {
	entities := msgpeer.NewEntities(nil, nil, nil)
	m := &tg.Message{
		ID:      100,
		Date:    int(time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC).Unix()),
		PeerID:  &tg.PeerUser{UserID: 7},
		Message: "hi",
	}
	m.SetEditDate(int(time.Date(2026, 4, 23, 13, 0, 0, 0, time.UTC).Unix()))
	m.SetGroupedID(1234567890)
	reactions := tg.MessageReactions{}
	rc1 := tg.ReactionCount{Reaction: &tg.ReactionEmoji{Emoticon: "👍"}, Count: 5}
	rc1.SetChosenOrder(0)
	rc2 := tg.ReactionCount{Reaction: &tg.ReactionCustomEmoji{DocumentID: 555}, Count: 1}
	reactions.Results = []tg.ReactionCount{rc1, rc2}
	m.SetReactions(reactions)

	row := messageToRow(m, entities, "@u")
	require.Equal(t, "2026-04-23T13:00:00Z", row.EditDate)
	require.Equal(t, int64(1234567890), row.GroupedID)
	require.Len(t, row.Reactions, 2)
	require.Equal(t, "emoji", row.Reactions[0].Kind)
	require.Equal(t, "👍", row.Reactions[0].Emoji)
	require.Equal(t, 5, row.Reactions[0].Count)
	require.True(t, row.Reactions[0].SelfReacted)
	require.Equal(t, "custom_emoji", row.Reactions[1].Kind)
	require.Equal(t, int64(555), row.Reactions[1].CustomEmojiID)
	require.False(t, row.Reactions[1].SelfReacted)
}

// TestMessageToRow_OmitsUnsetEditDateGroupedReactions asserts that a
// message without those optional fields produces a zero MessageRow
// triple — no spurious "1970-01-01" edit_date from a misread zero
// value, no GroupedID, no empty Reactions slice.
func TestMessageToRow_OmitsUnsetEditDateGroupedReactions(t *testing.T) {
	entities := msgpeer.NewEntities(nil, nil, nil)
	m := &tg.Message{
		ID:      101,
		Date:    int(time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC).Unix()),
		PeerID:  &tg.PeerUser{UserID: 7},
		Message: "hi",
	}
	row := messageToRow(m, entities, "@u")
	require.Empty(t, row.EditDate)
	require.Zero(t, row.GroupedID)
	require.Nil(t, row.Reactions)
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
