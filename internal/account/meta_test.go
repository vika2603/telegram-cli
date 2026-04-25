package account

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidName_good(t *testing.T) {
	// Cover the full allowed charset plus boundary first-char cases: letters
	// (both cases), digit, and each allowed interior separator (. _ -).
	for _, n := range []string{
		"a", "A", "0",
		"alice", "Alice_42",
		"work-1", "a.b_c", "A0",
		"alice.bot", "alice_bot", "alice-bot",
		"a.b.c.d", "x_y_z",
	} {
		require.True(t, IsValidName(n), n)
	}
}

func TestIsValidName_bad(t *testing.T) {
	// Reserved-name cases: "." and ".." must be rejected even though the
	// charset alone would accept them.
	reservedCases := []string{"", ".", ".."}

	// Leading-separator cases (regex requires [A-Za-z0-9] first).
	leadingCases := []string{"-leading", "_leading", ".leading"}

	// Every shell/OS metacharacter that could let a crafted name escape the
	// account directory sandbox or break quoting. We enumerate one case per
	// character to make a regression (e.g. someone widens the regex) obvious.
	metaChars := []string{
		"/", "\\", " ", "\t", "\n", "\r",
		"|", "&", ";", "$", "`", "\"", "'",
		"*", "?", "[", "]", "(", ")", "{", "}",
		"<", ">", "#", "~", "!", "@", "%", "^", "=", "+",
		",", ":",
	}
	var metaCases []string
	for _, c := range metaChars {
		metaCases = append(metaCases, "a"+c+"b") // interior
		metaCases = append(metaCases, c+"alice") // leading
		metaCases = append(metaCases, "alice"+c) // trailing
	}

	// Control chars (NUL + a low-range sample) and non-ASCII (multibyte UTF-8
	// + a single-byte high-bit char) must all be rejected.
	otherCases := []string{
		"\x00", string([]byte{0x01}), string([]byte{0x7f}),
		"日本", "café", string([]byte{0xff}),
	}

	for _, group := range [][]string{reservedCases, leadingCases, metaCases, otherCases} {
		for _, n := range group {
			require.False(t, IsValidName(n), "expected reject: %q", n)
		}
	}
}

func TestIsValidName_lengthLimit(t *testing.T) {
	require.True(t, IsValidName(string(bytes64('a'))))      // 64 chars OK
	require.False(t, IsValidName(string(bytes64('a'))+"a")) // 65 no
}

func bytes64(c byte) []byte {
	b := make([]byte, 64)
	for i := range b {
		b[i] = c
	}
	return b
}

func TestMeta_writeRead_roundtrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := Meta{
		Version:   1,
		Name:      "alice",
		State:     StateNEW,
		APIID:     12345,
		APIHash:   "hash",
		Phone:     "",
		CreatedAt: 1700000000,
	}
	require.NoError(t, WriteMeta(m))
	got, err := ReadMeta("alice")
	require.NoError(t, err)
	require.Equal(t, m, got)
	info, err := os.Stat(MetaFile("alice"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestReadMeta_missing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, err := ReadMeta("none")
	require.Error(t, err)
}

func TestReadMeta_wrongVersion(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := AccountDir("x")
	require.NoError(t, os.MkdirAll(dir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "account.json"),
		[]byte(`{"version":2,"name":"x"}`), 0600))
	_, err := ReadMeta("x")
	require.Error(t, err)
	require.Contains(t, err.Error(), "version")
}
