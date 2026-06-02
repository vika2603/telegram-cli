package message_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestPoll_NormalizesRegular(t *testing.T) {
	rows, err := actionmessage.Poll(context.Background(), actionmessage.PollRequest{
		RawRef:   "@grp",
		Question: "Lunch?",
		Options:  []string{"Pizza", "Sushi", ""},
		Public:   true,
	}, func(_ context.Context, q actionmessage.PollQuery) ([]output.SendResultRow, error) {
		require.Equal(t, "grp", q.Ref.Value)
		require.Equal(t, "Lunch?", q.Question)
		require.Equal(t, []string{"Pizza", "Sushi"}, q.Options) // blanks dropped
		require.True(t, q.Public)
		require.Equal(t, -1, q.CorrectIdx)
		return []output.SendResultRow{{Action: "poll", MessageID: 1}}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "poll", rows[0].Action)
}

func TestPoll_RequiresTwoOptions(t *testing.T) {
	_, err := actionmessage.Poll(context.Background(), actionmessage.PollRequest{
		RawRef: "@grp", Question: "Q", Options: []string{"only one"},
	}, func(context.Context, actionmessage.PollQuery) ([]output.SendResultRow, error) {
		t.Fatal("must not dispatch")
		return nil, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestPoll_QuizSetsCorrectIndex(t *testing.T) {
	_, err := actionmessage.Poll(context.Background(), actionmessage.PollRequest{
		RawRef: "@grp", Question: "2+2?", Options: []string{"3", "4", "5"},
		Correct: 2, Explanation: "basic math",
	}, func(_ context.Context, q actionmessage.PollQuery) ([]output.SendResultRow, error) {
		require.Equal(t, 1, q.CorrectIdx) // 1-based 2 -> 0-based 1
		require.Equal(t, "basic math", q.Explanation)
		return []output.SendResultRow{{Action: "poll"}}, nil
	})
	require.NoError(t, err)
}

func TestPoll_QuizRejectsMultiple(t *testing.T) {
	_, err := actionmessage.Poll(context.Background(), actionmessage.PollRequest{
		RawRef: "@grp", Question: "Q", Options: []string{"a", "b"}, Correct: 1, Multiple: true,
	}, func(context.Context, actionmessage.PollQuery) ([]output.SendResultRow, error) {
		t.Fatal("must not dispatch")
		return nil, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestPoll_CorrectOutOfRange(t *testing.T) {
	_, err := actionmessage.Poll(context.Background(), actionmessage.PollRequest{
		RawRef: "@grp", Question: "Q", Options: []string{"a", "b"}, Correct: 5,
	}, func(context.Context, actionmessage.PollQuery) ([]output.SendResultRow, error) {
		t.Fatal("must not dispatch")
		return nil, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestVote_OptionsAreZeroBased(t *testing.T) {
	_, err := actionmessage.Vote(context.Background(), actionmessage.VoteRequest{
		RawMessageRef: "@grp:42", Options: []int{1, 3},
	}, func(_ context.Context, q actionmessage.VoteQuery) (output.VoteResult, error) {
		require.Equal(t, "grp", q.Ref.Value)
		require.Equal(t, 42, q.MessageID)
		require.Equal(t, []int{0, 2}, q.OptionIdx)
		return output.VoteResult{Action: "vote", MessageID: 42}, nil
	})
	require.NoError(t, err)
}

func TestVote_Retract(t *testing.T) {
	_, err := actionmessage.Vote(context.Background(), actionmessage.VoteRequest{
		RawMessageRef: "@grp:42", Retract: true,
	}, func(_ context.Context, q actionmessage.VoteQuery) (output.VoteResult, error) {
		require.True(t, q.Retract)
		require.Empty(t, q.OptionIdx)
		return output.VoteResult{Action: "retract", MessageID: 42}, nil
	})
	require.NoError(t, err)
}

func TestVote_RequiresOptionsOrRetract(t *testing.T) {
	_, err := actionmessage.Vote(context.Background(), actionmessage.VoteRequest{
		RawMessageRef: "@grp:42",
	}, func(context.Context, actionmessage.VoteQuery) (output.VoteResult, error) {
		t.Fatal("must not dispatch")
		return output.VoteResult{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestVote_RetractRejectsOptions(t *testing.T) {
	_, err := actionmessage.Vote(context.Background(), actionmessage.VoteRequest{
		RawMessageRef: "@grp:42", Options: []int{1}, Retract: true,
	}, func(context.Context, actionmessage.VoteQuery) (output.VoteResult, error) {
		t.Fatal("must not dispatch")
		return output.VoteResult{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestVote_RejectsNonPositiveOption(t *testing.T) {
	_, err := actionmessage.Vote(context.Background(), actionmessage.VoteRequest{
		RawMessageRef: "@grp:42", Options: []int{0},
	}, func(context.Context, actionmessage.VoteQuery) (output.VoteResult, error) {
		t.Fatal("must not dispatch")
		return output.VoteResult{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestPoll_ExplanationRequiresQuiz(t *testing.T) {
	_, err := actionmessage.Poll(context.Background(), actionmessage.PollRequest{
		RawRef: "@grp", Question: "Q", Options: []string{"a", "b"}, Explanation: "x",
	}, func(context.Context, actionmessage.PollQuery) ([]output.SendResultRow, error) {
		t.Fatal("must not dispatch")
		return nil, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}
