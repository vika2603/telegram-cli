package message

import (
	"context"
	"fmt"
	"strings"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// maxPollOptions is Telegram's per-poll answer limit.
const maxPollOptions = 10

// PollRequest is the raw request for `tg msg poll`.
type PollRequest struct {
	RawRef      string
	Question    string
	Options     []string
	Multiple    bool
	Public      bool
	Correct     int    // 1-based correct option (quiz); 0 = regular poll
	Explanation string // quiz solution shown after answering
}

// PollQuery is the normalized payload passed to Telegram.
type PollQuery struct {
	Ref         ref.Ref
	Question    string
	Options     []string
	Multiple    bool
	Public      bool
	CorrectIdx  int // 0-based correct option (quiz); -1 = regular poll
	Explanation string
}

// PollFunc sends a poll.
type PollFunc func(context.Context, PollQuery) ([]output.SendResultRow, error)

// VoteRequest is the raw request for `tg msg vote`.
type VoteRequest struct {
	RawMessageRef string
	Options       []int // 1-based option numbers to vote for
	Retract       bool
}

// VoteQuery is the normalized payload passed to Telegram.
type VoteQuery struct {
	Ref       ref.Ref
	MessageID int
	OptionIdx []int // 0-based option indexes
	Retract   bool
}

// VoteFunc casts (or retracts) a vote on a poll.
type VoteFunc func(context.Context, VoteQuery) (output.VoteResult, error)

// Vote validates and dispatches `tg msg vote`. It only votes: pass option
// numbers to vote, or --retract to take a vote back.
func Vote(ctx context.Context, req VoteRequest, do VoteFunc) (output.VoteResult, error) {
	mref, err := parseMessageRef(req.RawMessageRef)
	if err != nil {
		return output.VoteResult{}, err
	}
	if req.Retract && len(req.Options) > 0 {
		return output.VoteResult{}, fmt.Errorf("%w: --retract takes no options", command.ErrUsage)
	}
	if !req.Retract && len(req.Options) == 0 {
		return output.VoteResult{}, fmt.Errorf("%w: give one or more option numbers, or --retract", command.ErrUsage)
	}
	idx := make([]int, 0, len(req.Options))
	for _, n := range req.Options {
		if n < 1 {
			return output.VoteResult{}, fmt.Errorf("%w: option numbers are 1-based (got %d)", command.ErrUsage, n)
		}
		idx = append(idx, n-1)
	}
	if do == nil {
		return output.VoteResult{}, fmt.Errorf("%w: msg vote called without vote function", command.ErrPrecondition)
	}
	return do(ctx, VoteQuery{
		Ref:       mref.Peer,
		MessageID: mref.MessageID,
		OptionIdx: idx,
		Retract:   req.Retract,
	})
}

// Poll validates and dispatches `tg msg poll`.
func Poll(ctx context.Context, req PollRequest, do PollFunc) ([]output.SendResultRow, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	question := strings.TrimSpace(req.Question)
	if question == "" {
		return nil, fmt.Errorf("%w: poll question is required", command.ErrUsage)
	}
	options := compactStrings(req.Options)
	if len(options) < 2 {
		return nil, fmt.Errorf("%w: a poll needs at least two options", command.ErrUsage)
	}
	if len(options) > maxPollOptions {
		return nil, fmt.Errorf("%w: a poll allows at most %d options", command.ErrUsage, maxPollOptions)
	}

	correctIdx := -1
	if req.Correct != 0 {
		if req.Multiple {
			return nil, fmt.Errorf("%w: a quiz (--correct) can't be multiple choice", command.ErrUsage)
		}
		if req.Correct < 1 || req.Correct > len(options) {
			return nil, fmt.Errorf("%w: --correct must be between 1 and %d", command.ErrUsage, len(options))
		}
		correctIdx = req.Correct - 1
	}
	if req.Explanation != "" && correctIdx < 0 {
		return nil, fmt.Errorf("%w: --explanation requires --correct (quiz mode)", command.ErrUsage)
	}
	if do == nil {
		return nil, fmt.Errorf("%w: msg poll called without poll function", command.ErrPrecondition)
	}
	return do(ctx, PollQuery{
		Ref:         parsed,
		Question:    question,
		Options:     options,
		Multiple:    req.Multiple,
		Public:      req.Public,
		CorrectIdx:  correctIdx,
		Explanation: req.Explanation,
	})
}
