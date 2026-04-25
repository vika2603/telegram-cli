package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMerge_higherOverridesLower(t *testing.T) {
	base := Config{DefaultAccount: ptr("a"), Output: OutputCfg{Format: ptr("human")}}
	over := Config{DefaultAccount: ptr("b")}
	got := Merge(base, over)
	require.Equal(t, "b", *got.DefaultAccount)
	require.Equal(t, "human", *got.Output.Format)
}

func TestMerge_unsetLeavesLower(t *testing.T) {
	base := Config{Output: OutputCfg{Format: ptr("human")}}
	over := Config{}
	got := Merge(base, over)
	require.Equal(t, "human", *got.Output.Format)
}

func TestMerge_aliasesReplaceNotUnion(t *testing.T) {
	base := Config{Aliases: map[string]string{"a": "1", "b": "2"}}
	over := Config{Aliases: map[string]string{"b": "X"}}
	got := Merge(base, over)
	require.Equal(t, map[string]string{"b": "X"}, got.Aliases)
}
