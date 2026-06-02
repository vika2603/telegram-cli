package telegram

import (
	"context"
	"fmt"

	gotdmessage "github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	telegrammessage "github.com/vika2603/telegram-cli/internal/telegram/message"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// SendPoll performs the RPC for `tg msg poll`.
func SendPoll(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionmessage.PollQuery) ([]output.SendResultRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return nil, err
	}

	answers := make([]gotdmessage.PollAnswerOption, len(q.Options))
	for i, opt := range q.Options {
		if i == q.CorrectIdx {
			answers[i] = gotdmessage.CorrectPollAnswer(opt)
		} else {
			answers[i] = gotdmessage.PollAnswer(opt)
		}
	}
	poll := gotdmessage.Poll(q.Question, answers[0], answers[1], answers[2:]...).
		PublicVoters(q.Public).
		MultipleChoice(q.Multiple)
	if q.Explanation != "" {
		poll = poll.Explanation(q.Explanation)
	}

	upd, err := gotdmessage.NewSender(api).To(resolved.InputPeer).Media(ctx, poll)
	if err != nil {
		return nil, err
	}
	return sentMessageRows("poll", upd), nil
}

// VotePoll casts (or retracts) a vote on a poll. It does not read back tallies
// — use `msg list` to see results.
func VotePoll(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionmessage.VoteQuery) (output.VoteResult, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.VoteResult{}, err
	}

	var options [][]byte
	if !q.Retract {
		// Fetch the poll once to map option numbers to their wire bytes.
		media, err := fetchPollMedia(ctx, api, resolved.InputPeer, q.MessageID)
		if err != nil {
			return output.VoteResult{}, err
		}
		if !media.Poll.MultipleChoice && len(q.OptionIdx) > 1 {
			return output.VoteResult{}, fmt.Errorf("%w: this poll is single-choice; pick exactly one option", command.ErrUsage)
		}
		options = make([][]byte, 0, len(q.OptionIdx))
		for _, i := range q.OptionIdx {
			if i < 0 || i >= len(media.Poll.Answers) {
				return output.VoteResult{}, fmt.Errorf("%w: option %d is out of range (poll has %d options)", command.ErrUsage, i+1, len(media.Poll.Answers))
			}
			options = append(options, media.Poll.Answers[i].Option)
		}
	}
	if _, err := api.MessagesSendVote(ctx, &tg.MessagesSendVoteRequest{
		Peer:    resolved.InputPeer,
		MsgID:   q.MessageID,
		Options: options,
	}); err != nil {
		return output.VoteResult{}, err
	}
	action := "vote"
	if q.Retract {
		action = "retract"
	}
	return output.VoteResult{Action: action, MessageID: q.MessageID}, nil
}

// MessageDetail reads a single message by ref and returns its full row,
// expanding poll content (question/options/tallies) when it is a poll.
func MessageDetail(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionmessage.InfoQuery) (output.MessageRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.MessageRow{}, err
	}
	elem, err := getMessageByID(ctx, api, resolved.InputPeer, q.MessageID)
	if err != nil {
		return output.MessageRow{}, err
	}
	m, ok := elem.Msg.(*tg.Message)
	if !ok {
		return output.MessageRow{}, telegrammessage.ErrNotFound
	}
	row := messageToRow(m, elem.Entities, output.PreferredRefFromResolved(resolved))
	if mp, ok := m.Media.(*tg.MessageMediaPoll); ok {
		pi := pollInfoFromMedia(mp)
		row.Poll = &pi
	}
	return row, nil
}

func pollInfoFromMedia(m *tg.MessageMediaPoll) output.PollInfo {
	info := output.PollInfo{
		Question:   m.Poll.Question.Text,
		Multiple:   m.Poll.MultipleChoice,
		Quiz:       m.Poll.Quiz,
		Public:     m.Poll.PublicVoters,
		Closed:     m.Poll.Closed,
		MinResults: m.Results.Min,
	}
	if tv, ok := m.Results.GetTotalVoters(); ok {
		info.TotalVoters = tv
	}
	tally := make(map[string]tg.PollAnswerVoters, len(m.Results.Results))
	for _, r := range m.Results.Results {
		tally[string(r.Option)] = r
	}
	for _, a := range m.Poll.Answers {
		opt := output.PollOption{Text: a.Text.Text}
		if r, ok := tally[string(a.Option)]; ok {
			opt.Voters = r.Voters
			opt.Chosen = r.Chosen
			opt.Correct = r.Correct
		}
		info.Options = append(info.Options, opt)
	}
	return info
}

func fetchPollMedia(ctx context.Context, api *tg.Client, inputPeer tg.InputPeerClass, msgID int) (*tg.MessageMediaPoll, error) {
	elem, err := getMessageByID(ctx, api, inputPeer, msgID)
	if err != nil {
		return nil, err
	}
	msg, ok := elem.Msg.(*tg.Message)
	if !ok {
		return nil, telegrammessage.ErrNotFound
	}
	media, ok := msg.Media.(*tg.MessageMediaPoll)
	if !ok {
		return nil, fmt.Errorf("%w: message %d is not a poll", command.ErrUsage, msgID)
	}
	return media, nil
}
