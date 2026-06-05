package contact_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/action/contact"
	"github.com/vika2603/telegram-cli/internal/command"
)

func TestReportRequiresFunction(t *testing.T) {
	err := contact.Report(context.Background(), contact.ReportRequest{RawRef: "@bob", Yes: true}, nil)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestReportDefaultsToSpam(t *testing.T) {
	err := contact.Report(context.Background(), contact.ReportRequest{RawRef: "@bob", Yes: true},
		func(_ context.Context, q contact.ReportQuery) error {
			require.Equal(t, "bob", q.Ref.Value)
			require.Equal(t, "spam", q.Reason)
			require.False(t, q.Block)
			return nil
		})
	require.NoError(t, err)
}

func TestReportPassesReasonMessageAndBlock(t *testing.T) {
	err := contact.Report(context.Background(), contact.ReportRequest{
		RawRef: "@bob", Reason: "fake", Message: "scam", Block: true, Yes: true,
	}, func(_ context.Context, q contact.ReportQuery) error {
		require.Equal(t, "fake", q.Reason)
		require.Equal(t, "scam", q.Message)
		require.True(t, q.Block)
		return nil
	})
	require.NoError(t, err)
}

func TestReportRejectsUnknownReason(t *testing.T) {
	err := contact.Report(context.Background(), contact.ReportRequest{RawRef: "@bob", Reason: "bogus", Yes: true},
		func(context.Context, contact.ReportQuery) error {
			t.Fatal("must not dispatch with an unknown reason")
			return nil
		})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestReportDeclined(t *testing.T) {
	called := false
	err := contact.Report(context.Background(), contact.ReportRequest{
		RawRef: "@bob", Prompter: stubPrompter{ok: false},
	}, func(context.Context, contact.ReportQuery) error {
		called = true
		return nil
	})
	require.ErrorIs(t, err, command.ErrNotConfirmed)
	require.False(t, called)
}
