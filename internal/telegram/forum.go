package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/gotd/td/crypto"
	"github.com/gotd/td/tg"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// ListForumTopics performs the RPC for `tg chat topics`.
func ListForumTopics(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.TopicsQuery) ([]output.TopicRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return nil, err
	}
	inCh, ok := inputChannelFromPeer(resolved.InputPeer)
	if !ok {
		return nil, fmt.Errorf("%w: topics are only available in forum supergroups", command.ErrUnsupported)
	}
	req := &tg.ChannelsGetForumTopicsRequest{Channel: inCh, Limit: q.Limit}
	if q.Q != "" {
		req.SetQ(q.Q)
	}
	resp, err := api.ChannelsGetForumTopics(ctx, req)
	if err != nil {
		return nil, mapForumErr(err)
	}
	rows := make([]output.TopicRow, 0, len(resp.Topics))
	for _, tc := range resp.Topics {
		// ForumTopicDeleted carries no fields worth surfacing; skip it.
		if t, ok := tc.(*tg.ForumTopic); ok {
			rows = append(rows, forumTopicToRow(t))
		}
	}
	return rows, nil
}

func forumTopicToRow(t *tg.ForumTopic) output.TopicRow {
	row := output.TopicRow{
		ID:          t.ID,
		Title:       t.Title,
		IconColor:   t.IconColor,
		TopMessage:  t.TopMessage,
		UnreadCount: t.UnreadCount,
		Closed:      t.Closed,
		Hidden:      t.Hidden,
		Pinned:      t.Pinned,
	}
	if v, ok := t.GetIconEmojiID(); ok {
		row.IconEmojiID = v
	}
	return row
}

// CreateForumTopic performs the RPC for `tg chat topics create`.
func CreateForumTopic(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.CreateTopicQuery) (output.TopicRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.TopicRow{}, err
	}
	inCh, ok := inputChannelFromPeer(resolved.InputPeer)
	if !ok {
		return output.TopicRow{}, fmt.Errorf("%w: topics can only be created in forum supergroups", command.ErrUnsupported)
	}
	// createForumTopic requires a random_id. Honor --random-id when given so
	// a retry dedupes; otherwise generate a fresh one.
	randomID := q.RandomID
	if randomID == 0 {
		if randomID, err = crypto.RandInt64(crypto.DefaultRand()); err != nil {
			return output.TopicRow{}, err
		}
	}
	req := &tg.ChannelsCreateForumTopicRequest{
		Channel:  inCh,
		Title:    q.Title,
		RandomID: randomID,
	}
	if q.IconColor != 0 {
		req.SetIconColor(q.IconColor)
	}
	if q.IconEmojiID != 0 {
		req.SetIconEmojiID(q.IconEmojiID)
	}
	upd, err := api.ChannelsCreateForumTopic(ctx, req)
	if err != nil {
		return output.TopicRow{}, mapForumErr(err)
	}
	row := output.TopicRow{Title: q.Title, IconColor: q.IconColor, IconEmojiID: q.IconEmojiID}
	// The topic id equals the id of the creation service message.
	if msgs := sentMessages(upd); len(msgs) > 0 {
		row.ID = msgs[0].MessageID
	}
	return row, nil
}

// mapForumErr translates "this chat is not a forum" RPC errors into a clear
// unsupported error; other errors pass through.
func mapForumErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "CHANNEL_FORUM_MISSING"),
		strings.Contains(msg, "FORUM_MISSING"):
		return fmt.Errorf("%w: this supergroup does not have topics enabled", command.ErrUnsupported)
	}
	return err
}
