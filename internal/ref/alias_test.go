package ref

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpandAlias_hit(t *testing.T) {
	aliases := map[string]string{"boss": "@alice"}
	require.Equal(t, "@alice", ExpandAlias("boss", aliases))
}

func TestExpandAlias_miss_passthrough(t *testing.T) {
	aliases := map[string]string{"boss": "@alice"}
	require.Equal(t, "@bob", ExpandAlias("@bob", aliases))
}

func TestExpandAlias_single_level_no_recursion(t *testing.T) {
	aliases := map[string]string{"a": "b", "b": "c"}
	require.Equal(t, "b", ExpandAlias("a", aliases))
}

func TestExpandAlias_nil_map(t *testing.T) {
	require.Equal(t, "@bob", ExpandAlias("@bob", nil))
}
