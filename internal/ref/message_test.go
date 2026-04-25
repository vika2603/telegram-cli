package ref

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMessageRef(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantID  int
		wantRef string
	}{
		{"@news:42", "news", 42, "@news:42"},
		{"g:77:10", "chat", 10, "g:77:10"},
		{"u:42:9001:7", "user", 7, "u:42:9001:7"},
		{"c:100:555:9", "channel", 9, "c:100:555:9"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseMessageRef(tc.in)
			require.NoError(t, err)
			require.Equal(t, tc.want, got.Peer.Value)
			require.Equal(t, tc.wantID, got.MessageID)
			require.Equal(t, tc.wantRef, got.String())
		})
	}
}

func TestParseMessageRefRejectsMalformed(t *testing.T) {
	for _, in := range []string{"@news", "@news:0", "@news:nope", ":1", "u:1:2"} {
		t.Run(in, func(t *testing.T) {
			_, err := ParseMessageRef(in)
			require.Error(t, err)
		})
	}
}
