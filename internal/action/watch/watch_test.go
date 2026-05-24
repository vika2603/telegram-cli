package watch_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	actionwatch "github.com/vika2603/telegram-cli/internal/action/watch"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/ref"
)

func TestNormalize_AcceptsEmptyRefsAndKinds(t *testing.T) {
	q, err := actionwatch.Normalize(actionwatch.Request{})
	require.NoError(t, err)
	require.Empty(t, q.Refs)
	require.Empty(t, q.Kinds)
	require.Equal(t, 0, q.Limit)
}

func TestNormalize_ParsesValidRefsAndKinds(t *testing.T) {
	q, err := actionwatch.Normalize(actionwatch.Request{
		RawRefs: []string{"@chan", "me"},
		Kinds:   []string{"message", "edit"},
		Limit:   5,
	})
	require.NoError(t, err)
	require.Len(t, q.Refs, 2)
	require.Equal(t, ref.RefKindUsername, q.Refs[0].Kind)
	require.Equal(t, "chan", q.Refs[0].Value)
	require.Equal(t, ref.RefKindMe, q.Refs[1].Kind)
	require.Equal(t, []string{"message", "edit"}, q.Kinds)
	require.Equal(t, 5, q.Limit)
}

func TestNormalize_RejectsNegativeLimit(t *testing.T) {
	_, err := actionwatch.Normalize(actionwatch.Request{Limit: -1})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestNormalize_RejectsUnknownKind(t *testing.T) {
	_, err := actionwatch.Normalize(actionwatch.Request{Kinds: []string{"reaction"}})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestNormalize_RejectsMalformedRef(t *testing.T) {
	_, err := actionwatch.Normalize(actionwatch.Request{RawRefs: []string{""}})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_DelegatesToStream(t *testing.T) {
	var got actionwatch.Query
	called := 0
	stream := func(_ context.Context, q actionwatch.Query) error {
		called++
		got = q
		return nil
	}
	err := actionwatch.Run(context.Background(), actionwatch.Request{
		RawRefs: []string{"@chan"}, Limit: 3,
	}, stream)
	require.NoError(t, err)
	require.Equal(t, 1, called)
	require.Len(t, got.Refs, 1)
	require.Equal(t, 3, got.Limit)
}

func TestRun_PropagatesStreamError(t *testing.T) {
	want := errors.New("boom")
	err := actionwatch.Run(context.Background(), actionwatch.Request{},
		func(context.Context, actionwatch.Query) error { return want })
	require.ErrorIs(t, err, want)
}

func TestRun_NilStreamReturnsPrecondition(t *testing.T) {
	err := actionwatch.Run(context.Background(), actionwatch.Request{}, nil)
	require.ErrorIs(t, err, command.ErrPrecondition)
}
