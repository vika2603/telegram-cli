package profile

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
)

func TestSetBioReadsStdin(t *testing.T) {
	row, err := SetBio(context.Background(), SetBioRequest{
		Bio:   "-",
		Stdin: strings.NewReader("hello\n"),
	}, func(_ context.Context, bio string) (output.ProfileRow, error) {
		return output.ProfileRow{Action: "set-bio", Bio: bio}, nil
	})

	require.NoError(t, err)
	require.Equal(t, "hello", row.Bio)
}
