package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromEnv_goodValues(t *testing.T) {
	t.Setenv("TG_API_ID", "12345")
	t.Setenv("TG_API_HASH", "hash")
	t.Setenv("TG_FLOOD_WAIT_MAX", "60")
	c, err := FromEnv()
	require.NoError(t, err)
	require.NotNil(t, c.APIID)
	require.Equal(t, 12345, *c.APIID)
	require.NotNil(t, c.FloodWait.MaxSeconds)
	require.Equal(t, 60, *c.FloodWait.MaxSeconds)
}

func TestFromEnv_emptyIntsIgnored(t *testing.T) {
	t.Setenv("TG_API_ID", "")
	t.Setenv("TG_FLOOD_WAIT_MAX", "")
	c, err := FromEnv()
	require.NoError(t, err)
	require.Nil(t, c.APIID)
	require.Nil(t, c.FloodWait.MaxSeconds)
}

func TestFromEnv_malformedAPIIDIsErrUsage(t *testing.T) {
	t.Setenv("TG_API_ID", "not-a-number")
	_, err := FromEnv()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalid)
	require.Contains(t, err.Error(), "TG_API_ID")
}

func TestFromEnv_malformedFloodMaxIsErrUsage(t *testing.T) {
	t.Setenv("TG_FLOOD_WAIT_MAX", "abc")
	_, err := FromEnv()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalid)
	require.Contains(t, err.Error(), "TG_FLOOD_WAIT_MAX")
}

func TestFromEnv_negativeFloodMaxIsErrUsage(t *testing.T) {
	t.Setenv("TG_FLOOD_WAIT_MAX", "-5")
	_, err := FromEnv()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalid)
}
