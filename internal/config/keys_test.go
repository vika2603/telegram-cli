package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveKey_staticKeys(t *testing.T) {
	for _, k := range Keys {
		got, err := ResolveKey(k.Name)
		require.NoError(t, err, "ResolveKey(%q)", k.Name)
		require.Equal(t, k.Name, got.Name)
		require.Equal(t, k.Type, got.Type)
	}
}

func TestResolveKey_alias(t *testing.T) {
	got, err := ResolveKey("aliases.work")
	require.NoError(t, err)
	require.Equal(t, KeyAlias, got.Type)
	require.Equal(t, "aliases.work", got.Name)
}

func TestResolveKey_aliasEmptySubkey(t *testing.T) {
	_, err := ResolveKey("aliases.")
	require.Error(t, err)
	require.Contains(t, err.Error(), "subkey is required")
}

func TestResolveKey_unknown(t *testing.T) {
	_, err := ResolveKey("nope")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown config key")
	require.Contains(t, err.Error(), "valid keys:")
	require.Contains(t, err.Error(), "aliases.<name>")
}

func TestCoerceValue_string(t *testing.T) {
	k, _ := ResolveKey("default_account")
	v, err := CoerceValue(k, "myacct")
	require.NoError(t, err)
	require.Equal(t, "myacct", v)
}

func TestCoerceValue_int_valid(t *testing.T) {
	k, _ := ResolveKey("api_id")
	v, err := CoerceValue(k, "12345")
	require.NoError(t, err)
	require.Equal(t, int64(12345), v)
}

func TestCoerceValue_int_invalid(t *testing.T) {
	k, _ := ResolveKey("api_id")
	_, err := CoerceValue(k, "abc")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not an integer")
}

func TestCoerceValue_int_nonPositiveAPIID(t *testing.T) {
	k, _ := ResolveKey("api_id")
	_, err := CoerceValue(k, "0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be positive")

	_, err = CoerceValue(k, "-1")
	require.Error(t, err)
}

func TestCoerceValue_int_negativeFloodWait(t *testing.T) {
	k, _ := ResolveKey("flood_wait.max_seconds")
	_, err := CoerceValue(k, "-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be non-negative")

	v, err := CoerceValue(k, "0")
	require.NoError(t, err)
	require.Equal(t, int64(0), v)
}

func TestCoerceValue_enum_valid(t *testing.T) {
	k, _ := ResolveKey("output.format")
	v, err := CoerceValue(k, "json")
	require.NoError(t, err)
	require.Equal(t, "json", v)
}

func TestCoerceValue_enum_invalid(t *testing.T) {
	k, _ := ResolveKey("output.format")
	_, err := CoerceValue(k, "yaml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be one of")
}

func TestCoerceValue_alias(t *testing.T) {
	k, _ := ResolveKey("aliases.boss")
	v, err := CoerceValue(k, "@foo")
	require.NoError(t, err)
	require.Equal(t, "@foo", v)
}

func TestReadWriteUnsetRaw_roundTrip(t *testing.T) {
	raw := map[string]any{}

	// write a nested key
	WriteRaw(raw, "output.format", "json")
	v, ok := ReadRaw(raw, "output.format")
	require.True(t, ok)
	require.Equal(t, "json", v)

	// overwrite
	WriteRaw(raw, "output.format", "human")
	v, ok = ReadRaw(raw, "output.format")
	require.True(t, ok)
	require.Equal(t, "human", v)

	// write a top-level key
	WriteRaw(raw, "default_account", "main")
	v, ok = ReadRaw(raw, "default_account")
	require.True(t, ok)
	require.Equal(t, "main", v)

	// unset it
	removed := UnsetRaw(raw, "default_account")
	require.True(t, removed)
	_, ok = ReadRaw(raw, "default_account")
	require.False(t, ok)

	// unset missing key returns false
	removed = UnsetRaw(raw, "default_account")
	require.False(t, removed)

	// unset nested
	removed = UnsetRaw(raw, "output.format")
	require.True(t, removed)
	_, ok = ReadRaw(raw, "output.format")
	require.False(t, ok)
}

func TestReadRaw_missingPath(t *testing.T) {
	raw := map[string]any{}
	_, ok := ReadRaw(raw, "output.format")
	require.False(t, ok)

	_, ok = ReadRaw(raw, "nonexistent")
	require.False(t, ok)
}

func TestReadRaw_deepNesting(t *testing.T) {
	raw := map[string]any{}
	WriteRaw(raw, "flood_wait.max_seconds", int64(60))
	v, ok := ReadRaw(raw, "flood_wait.max_seconds")
	require.True(t, ok)
	require.Equal(t, int64(60), v)
}

// TestKeys_enumValuesPopulated ensures every KeyEnum has at least one value.
func TestKeys_enumValuesPopulated(t *testing.T) {
	for _, k := range Keys {
		if k.Type == KeyEnum {
			require.NotEmpty(t, k.EnumValues, "key %q has KeyEnum but no EnumValues", k.Name)
		}
	}
}

// TestResolveKey_aliasPrefix verifies that keys starting with "aliases." but
// with non-empty subkeys always succeed.
func TestResolveKey_aliasPrefix(t *testing.T) {
	cases := []string{"aliases.a", "aliases.boss", "aliases.work-account"}
	for _, name := range cases {
		got, err := ResolveKey(name)
		require.NoError(t, err, "ResolveKey(%q)", name)
		require.Equal(t, KeyAlias, got.Type)
		require.True(t, strings.HasPrefix(got.Name, "aliases."))
	}
}
