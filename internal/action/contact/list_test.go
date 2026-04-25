package contact_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/action/contact"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestListRequiresFetch(t *testing.T) {
	_, err := contact.List(context.Background(), contact.ListRequest{}, nil)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestListNormalizesRequest(t *testing.T) {
	rows, err := contact.List(context.Background(), contact.ListRequest{
		Blocked:    true,
		MutualOnly: true,
		Bots:       true,
	}, func(_ context.Context, q contact.ListQuery) ([]output.ContactRow, error) {
		require.True(t, q.Blocked)
		require.True(t, q.MutualOnly)
		require.True(t, q.Bots)
		return []output.ContactRow{{ID: 1}}, nil
	})
	require.NoError(t, err)
	require.Equal(t, []output.ContactRow{{ID: 1}}, rows)
}
