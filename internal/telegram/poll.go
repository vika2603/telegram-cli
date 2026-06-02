package telegram

import (
	"context"

	gotdmessage "github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/output"
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
