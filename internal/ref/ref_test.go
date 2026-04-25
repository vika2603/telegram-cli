package ref

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRef_me(t *testing.T) {
	r, err := ParseRef("me")
	require.NoError(t, err)
	require.Equal(t, RefKindMe, r.Kind)
}

func TestParseRef_saved(t *testing.T) {
	r, err := ParseRef("saved")
	require.NoError(t, err)
	require.Equal(t, RefKindSaved, r.Kind)
}

func TestParseRef_username(t *testing.T) {
	r, err := ParseRef("@alice")
	require.NoError(t, err)
	require.Equal(t, RefKindUsername, r.Kind)
	require.Equal(t, "alice", r.Value)
}

func TestParseRef_username_rejects_bare_at(t *testing.T) {
	_, err := ParseRef("@")
	require.Error(t, err)
}

func TestParseRef_phone(t *testing.T) {
	r, err := ParseRef("+8613800138000")
	require.NoError(t, err)
	require.Equal(t, RefKindPhone, r.Kind)
	require.Equal(t, "8613800138000", r.Value)
}

func TestParseRef_phone_rejects_letters(t *testing.T) {
	_, err := ParseRef("+86abc")
	require.Error(t, err)
}

func TestParseRef_id_positive(t *testing.T) {
	r, err := ParseRef("1234567")
	require.NoError(t, err)
	require.Equal(t, RefKindID, r.Kind)
	require.Equal(t, int64(1234567), r.ID)
}

func TestParseRef_id_negative_channel(t *testing.T) {
	r, err := ParseRef("-1001234567890")
	require.NoError(t, err)
	require.Equal(t, RefKindID, r.Kind)
	require.Equal(t, int64(-1001234567890), r.ID)
}

func TestParseRef_DirectPeerRefs(t *testing.T) {
	cases := []struct {
		in         string
		wantKind   string
		wantID     int64
		wantHash   int64
		wantString string
	}{
		{"u:42:9001", "user", 42, 9001, "u:42:9001"},
		{"user:42:9001", "user", 42, 9001, "u:42:9001"},
		{"g:77", "chat", 77, 0, "g:77"},
		{"chat:77", "chat", 77, 0, "g:77"},
		{"c:100:555", "channel", 100, 555, "c:100:555"},
		{"channel:100:555", "channel", 100, 555, "c:100:555"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseRef(tc.in)
			require.NoError(t, err)
			require.Equal(t, RefKindPeer, got.Kind)
			require.Equal(t, tc.wantKind, got.Value)
			require.Equal(t, tc.wantID, got.ID)
			require.Equal(t, tc.wantHash, got.AccessHash)
			require.Equal(t, tc.wantString, got.String())
		})
	}
}

func TestParseRef_DirectPeerRefsRejectMalformed(t *testing.T) {
	for _, in := range []string{"u:42", "c:abc:1", "g:-1", "g:1:2"} {
		t.Run(in, func(t *testing.T) {
			_, err := ParseRef(in)
			require.Error(t, err)
		})
	}
}

func TestParseRef_empty_rejected(t *testing.T) {
	_, err := ParseRef("")
	require.Error(t, err)
}

func TestParseRef_TMeLink(t *testing.T) {
	cases := []struct {
		in       string
		wantKind RefKind
		wantVal  string
	}{
		{"t.me/durov", RefKindTMeLink, "durov"},
		{"https://t.me/durov", RefKindTMeLink, "durov"},
		{"http://t.me/joinchat/ABC", RefKindTMeLink, "joinchat/ABC"},
		{"t.me/c/1234567890/42", RefKindTMeLink, "c/1234567890/42"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseRef(tc.in)
			require.NoError(t, err)
			require.Equal(t, tc.wantKind, got.Kind)
			require.Equal(t, tc.wantVal, got.Value)
		})
	}
}

func TestParseRef_TgDeeplink(t *testing.T) {
	cases := []struct {
		in      string
		wantVal string
	}{
		{"tg://resolve?domain=durov", "resolve?domain=durov"},
		{"tg://join?invite=ABC", "join?invite=ABC"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseRef(tc.in)
			require.NoError(t, err)
			require.Equal(t, RefKindTGDeeplink, got.Kind)
			require.Equal(t, tc.wantVal, got.Value)
		})
	}
}

func TestIsInviteLink(t *testing.T) {
	cases := []struct {
		name     string
		ref      Ref
		want     bool
		wantHash string
	}{
		{
			name:     "plus_hash",
			ref:      Ref{Kind: RefKindTMeLink, Value: "+abc123"},
			want:     true,
			wantHash: "abc123",
		},
		{
			name:     "joinchat_prefix",
			ref:      Ref{Kind: RefKindTMeLink, Value: "joinchat/abc123"},
			want:     true,
			wantHash: "abc123",
		},
		{
			name: "plain_username_tme",
			ref:  Ref{Kind: RefKindTMeLink, Value: "durov"},
			want: false,
		},
		{
			name: "username_kind",
			ref:  Ref{Kind: RefKindUsername, Value: "durov"},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.ref.IsInviteLink())
			if tc.want {
				require.Equal(t, tc.wantHash, tc.ref.InviteHash())
			}
		})
	}
}

func TestParseRef_TMeLinkRejectsEmpty(t *testing.T) {
	_, err := ParseRef("t.me/")
	require.Error(t, err)
	_, err = ParseRef("https://t.me/")
	require.Error(t, err)
	_, err = ParseRef("tg://")
	require.Error(t, err)
}
