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
	"github.com/vika2603/telegram-cli/internal/ref"
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

	const batch = 100
	var rows []output.TopicRow
	var offsetDate, offsetID, offsetTopic int
	for len(rows) < q.Limit {
		limit := batch
		if rem := q.Limit - len(rows); rem < limit {
			limit = rem
		}
		req := &tg.ChannelsGetForumTopicsRequest{
			Channel:     inCh,
			Limit:       limit,
			OffsetDate:  offsetDate,
			OffsetID:    offsetID,
			OffsetTopic: offsetTopic,
		}
		if q.Q != "" {
			req.SetQ(q.Q)
		}
		resp, err := api.ChannelsGetForumTopics(ctx, req)
		if err != nil {
			return nil, mapForumErr(err)
		}
		if len(resp.Topics) == 0 {
			break
		}
		var last *tg.ForumTopic
		for _, tc := range resp.Topics {
			// ForumTopicDeleted carries no fields worth surfacing; skip it.
			if t, ok := tc.(*tg.ForumTopic); ok {
				rows = append(rows, forumTopicToRow(t))
				last = t
			}
		}
		// Stop when the server returned fewer than requested (last page) or we
		// can't derive a pagination cursor from this page.
		if len(resp.Topics) < limit || last == nil {
			break
		}
		offsetTopic = last.ID
		offsetID = last.TopMessage
		if resp.OrderByCreateDate {
			offsetDate = last.Date
		} else {
			offsetDate = forumMessageDate(resp.Messages, last.TopMessage)
		}
	}
	return rows, nil
}

// forumMessageDate finds the date of the message with the given id among a
// forum-topics response's related messages, for offset-based pagination.
func forumMessageDate(msgs []tg.MessageClass, id int) int {
	for _, m := range msgs {
		switch v := m.(type) {
		case *tg.Message:
			if v.ID == id {
				return v.Date
			}
		case *tg.MessageService:
			if v.ID == id {
				return v.Date
			}
		}
	}
	return 0
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
	// The topic id equals the id of the creation service message.
	msgs := sentMessages(upd)
	if len(msgs) == 0 {
		return output.TopicRow{}, fmt.Errorf("create topic %q: server response carried no topic id", q.Title)
	}
	return output.TopicRow{ID: msgs[0].MessageID, Title: q.Title, IconColor: q.IconColor, IconEmojiID: q.IconEmojiID}, nil
}

// EditForumTopic performs the RPC for `tg chat topics edit`. Only the fields
// the caller set (non-nil) are sent, so an edit changes title/closed/hidden
// independently.
func EditForumTopic(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.EditTopicQuery) (output.TopicRow, error) {
	inCh, err := topicChannel(ctx, resolver, q.Ref)
	if err != nil {
		return output.TopicRow{}, err
	}
	req := &tg.ChannelsEditForumTopicRequest{Channel: inCh, TopicID: q.TopicID}
	row := output.TopicRow{ID: q.TopicID}
	if q.Title != nil {
		req.SetTitle(*q.Title)
		row.Title = *q.Title
	}
	if q.Closed != nil {
		req.SetClosed(*q.Closed)
		row.Closed = *q.Closed
	}
	if q.Hidden != nil {
		req.SetHidden(*q.Hidden)
		row.Hidden = *q.Hidden
	}
	if _, err := api.ChannelsEditForumTopic(ctx, req); err != nil {
		return output.TopicRow{}, mapForumErr(err)
	}
	return row, nil
}

// DeleteForumTopic performs the RPC for `tg chat topics delete`. It deletes
// the topic's message history; deleteTopicHistory returns the remaining
// offset, so loop (bounded) until it's drained.
func DeleteForumTopic(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.DeleteTopicQuery) error {
	inCh, err := topicChannel(ctx, resolver, q.Ref)
	if err != nil {
		return err
	}
	for range 100 {
		aff, err := api.ChannelsDeleteTopicHistory(ctx, &tg.ChannelsDeleteTopicHistoryRequest{
			Channel:  inCh,
			TopMsgID: q.TopicID,
		})
		if err != nil {
			return mapForumErr(err)
		}
		if aff.Offset <= 0 {
			return nil
		}
	}
	return fmt.Errorf("delete topic %d: history not fully drained after 100 batches; rerun to continue", q.TopicID)
}

// PinForumTopic performs the RPC for `tg chat topics pin`.
func PinForumTopic(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.PinTopicQuery) (output.TopicRow, error) {
	inCh, err := topicChannel(ctx, resolver, q.Ref)
	if err != nil {
		return output.TopicRow{}, err
	}
	if _, err := api.ChannelsUpdatePinnedForumTopic(ctx, &tg.ChannelsUpdatePinnedForumTopicRequest{
		Channel: inCh,
		TopicID: q.TopicID,
		Pinned:  q.Pinned,
	}); err != nil {
		return output.TopicRow{}, mapForumErr(err)
	}
	return output.TopicRow{ID: q.TopicID, Pinned: q.Pinned}, nil
}

// GetForumTopicByID fetches a single forum topic by its ID.
func GetForumTopicByID(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.TopicInfoQuery) (output.TopicRow, error) {
	inCh, err := topicChannel(ctx, resolver, q.Ref)
	if err != nil {
		return output.TopicRow{}, err
	}
	resp, err := api.ChannelsGetForumTopicsByID(ctx, &tg.ChannelsGetForumTopicsByIDRequest{
		Channel: inCh,
		Topics:  []int{q.TopicID},
	})
	if err != nil {
		return output.TopicRow{}, mapForumErr(err)
	}
	for _, tc := range resp.Topics {
		if t, ok := tc.(*tg.ForumTopic); ok && t.ID == q.TopicID {
			return forumTopicToRow(t), nil
		}
	}
	return output.TopicRow{}, fmt.Errorf("%w: topic %d", peer.ErrNotFound, q.TopicID)
}

// MuteForumTopic mutes or unmutes a single forum topic's notifications.
func MuteForumTopic(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.MuteTopicQuery) (output.TopicRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.TopicRow{}, err
	}
	if _, ok := inputChannelFromPeer(resolved.InputPeer); !ok {
		return output.TopicRow{}, fmt.Errorf("%w: topics are only available in forum supergroups", command.ErrUnsupported)
	}
	settings := &tg.InputPeerNotifySettings{}
	settings.SetMuteUntil(q.MuteUntil)
	req := &tg.AccountUpdateNotifySettingsRequest{
		Peer:     &tg.InputNotifyForumTopic{Peer: resolved.InputPeer, TopMsgID: q.TopicID},
		Settings: *settings,
	}
	if _, err := api.AccountUpdateNotifySettings(ctx, req); err != nil {
		return output.TopicRow{}, mapForumErr(err)
	}
	return output.TopicRow{ID: q.TopicID}, nil
}

// ReadForumTopic marks a forum topic as read up to its latest message.
func ReadForumTopic(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.ReadTopicQuery) (output.TopicRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.TopicRow{}, err
	}
	inCh, ok := inputChannelFromPeer(resolved.InputPeer)
	if !ok {
		return output.TopicRow{}, fmt.Errorf("%w: topics are only available in forum supergroups", command.ErrUnsupported)
	}
	resp, err := api.ChannelsGetForumTopicsByID(ctx, &tg.ChannelsGetForumTopicsByIDRequest{
		Channel: inCh,
		Topics:  []int{q.TopicID},
	})
	if err != nil {
		return output.TopicRow{}, mapForumErr(err)
	}
	topMessage := 0
	for _, tc := range resp.Topics {
		if t, ok := tc.(*tg.ForumTopic); ok && t.ID == q.TopicID {
			topMessage = t.TopMessage
			break
		}
	}
	if topMessage == 0 {
		return output.TopicRow{}, fmt.Errorf("%w: topic %d", peer.ErrNotFound, q.TopicID)
	}
	if _, err := api.MessagesReadDiscussion(ctx, &tg.MessagesReadDiscussionRequest{
		Peer:      resolved.InputPeer,
		MsgID:     q.TopicID,
		ReadMaxID: topMessage,
	}); err != nil {
		return output.TopicRow{}, mapForumErr(err)
	}
	return output.TopicRow{ID: q.TopicID}, nil
}

// topicChannel resolves a ref to the InputChannel forum RPCs need.
func topicChannel(ctx context.Context, resolver *peer.Resolver, target ref.Ref) (tg.InputChannelClass, error) {
	resolved, err := resolver.Resolve(ctx, target)
	if err != nil {
		return nil, err
	}
	inCh, ok := inputChannelFromPeer(resolved.InputPeer)
	if !ok {
		return nil, fmt.Errorf("%w: topics are only available in forum supergroups", command.ErrUnsupported)
	}
	return inCh, nil
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
