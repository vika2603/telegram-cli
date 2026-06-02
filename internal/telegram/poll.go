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

// VotePoll reads a poll and optionally casts (or retracts) a vote, returning
// the poll's current state.
func VotePoll(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionmessage.VoteQuery) (output.PollInfo, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.PollInfo{}, err
	}
	media, err := fetchPollMedia(ctx, api, resolved.InputPeer, q.MessageID)
	if err != nil {
		return output.PollInfo{}, err
	}
	if q.Show {
		return pollInfoFromMedia(media), nil
	}
	if !q.Retract && !media.Poll.MultipleChoice && len(q.OptionIdx) > 1 {
		return output.PollInfo{}, fmt.Errorf("%w: this poll is single-choice; pick exactly one option", command.ErrUsage)
	}

	options := make([][]byte, 0, len(q.OptionIdx))
	if !q.Retract {
		for _, i := range q.OptionIdx {
			if i < 0 || i >= len(media.Poll.Answers) {
				return output.PollInfo{}, fmt.Errorf("%w: option %d is out of range (poll has %d options)", command.ErrUsage, i+1, len(media.Poll.Answers))
			}
			options = append(options, media.Poll.Answers[i].Option)
		}
	}
	if _, err := api.MessagesSendVote(ctx, &tg.MessagesSendVoteRequest{
		Peer:    resolved.InputPeer,
		MsgID:   q.MessageID,
		Options: options,
	}); err != nil {
		return output.PollInfo{}, err
	}
	// Re-fetch for the updated tallies.
	fresh, err := fetchPollMedia(ctx, api, resolved.InputPeer, q.MessageID)
	if err != nil {
		return output.PollInfo{}, err
	}
	return pollInfoFromMedia(fresh), nil
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

func pollInfoFromMedia(m *tg.MessageMediaPoll) output.PollInfo {
	info := output.PollInfo{
		Question: m.Poll.Question.Text,
		Multiple: m.Poll.MultipleChoice,
		Quiz:     m.Poll.Quiz,
		Public:   m.Poll.PublicVoters,
		Closed:   m.Poll.Closed,
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
