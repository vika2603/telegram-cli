package telegram

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsBadPassword(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"rpc error: code 400 desc = PASSWORD_HASH_INVALID (hash does not match)", true},
		{"rpc error: code 400 desc = SRP_PASSWORD_CHANGED", true},
		{"rpc error: code 400 desc = PASSWORD_MISSING", true},
		{"rpc error: code 500 desc = INTERNAL_SERVER_ERROR", false},
		{"rpc error: code 420 desc = FLOOD_WAIT_60", false},
		{"", false},
	}
	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			require.Equal(t, c.want, IsBadPassword(errors.New(c.msg)))
		})
	}
}
