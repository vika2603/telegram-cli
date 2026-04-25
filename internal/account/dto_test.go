package account

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDTOFromMeta_fullMetaKeepsFields(t *testing.T) {
	m := Meta{
		Name: "alice", State: StateAUTHED,
		APIID: 12345, APIHash: "h",
		Phone: "+15551234567", CreatedAt: 1700000000,
	}
	d := DTOFromMeta(m, true)
	require.Equal(t, "alice", d.Name)
	require.Equal(t, "AUTHED", d.State)
	require.NotNil(t, d.Phone)
	require.Equal(t, "+15551234567", *d.Phone)
	require.NotNil(t, d.APIID)
	require.Equal(t, 12345, *d.APIID)
	require.EqualValues(t, 1700000000, d.CreatedAt)
	require.True(t, d.Default)
}

func TestDTOFromMeta_bareSlotOmitsPhoneAndAPIID(t *testing.T) {
	m := Meta{Name: "alice", State: StateNEW, CreatedAt: 1}
	d := DTOFromMeta(m, false)
	require.Nil(t, d.Phone)
	require.Nil(t, d.APIID)
	require.False(t, d.Default)
	b, err := json.Marshal(d)
	require.NoError(t, err)
	s := string(b)
	require.NotContains(t, s, "phone")
	require.NotContains(t, s, "api_id")
}

func TestDTOFromMeta_jsonContainsRequiredKeys(t *testing.T) {
	m := Meta{Name: "bob", State: StateNEW, CreatedAt: 2}
	b, err := json.Marshal(DTOFromMeta(m, false))
	require.NoError(t, err)
	var obj map[string]any
	require.NoError(t, json.Unmarshal(b, &obj))
	require.Equal(t, "bob", obj["name"])
	require.Equal(t, "NEW", obj["state"])
	require.Contains(t, obj, "created_at")
	require.Contains(t, obj, "default")
}

func TestAccountDTO_humanDefaultMarker(t *testing.T) {
	d := DTOFromMeta(Meta{Name: "alice", State: StateAUTHED}, true)
	require.True(t, strings.HasPrefix(d.Human(), "*"),
		"default row must start with *: %q", d.Human())
	d2 := DTOFromMeta(Meta{Name: "alice", State: StateAUTHED}, false)
	require.False(t, strings.HasPrefix(d2.Human(), "*"))
	require.Contains(t, d.Human(), "alice")
	require.Contains(t, d.Human(), "AUTHED")
}
