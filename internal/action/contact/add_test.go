package contact_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/action/contact"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestAddRequiresFunction(t *testing.T) {
	_, err := contact.Add(context.Background(), contact.AddRequest{Phone: "+1", First: "Alice"}, nil)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestAddRequiresPhone(t *testing.T) {
	_, err := contact.Add(context.Background(), contact.AddRequest{First: "Alice"}, func(context.Context, contact.AddQuery) (output.ContactRow, error) {
		t.Fatal("add must not run without phone")
		return output.ContactRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestAddRequiresFirstName(t *testing.T) {
	_, err := contact.Add(context.Background(), contact.AddRequest{Phone: "+1"}, func(context.Context, contact.AddQuery) (output.ContactRow, error) {
		t.Fatal("add must not run without first name")
		return output.ContactRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestAddNormalizesRequest(t *testing.T) {
	row, err := contact.Add(context.Background(), contact.AddRequest{
		Phone: "+1", First: "Alice", Last: "Smith", Mutual: true,
	}, func(_ context.Context, q contact.AddQuery) (output.ContactRow, error) {
		require.Equal(t, "+1", q.Phone)
		require.Equal(t, "Alice", q.First)
		require.Equal(t, "Smith", q.Last)
		require.True(t, q.Mutual)
		return output.ContactRow{ID: 10}, nil
	})
	require.NoError(t, err)
	require.Equal(t, output.ContactRow{ID: 10}, row)
}
