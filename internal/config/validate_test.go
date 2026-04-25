package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func strp(s string) *string { return &s }
func intp(i int) *int       { return &i }

func TestValidateEnums_allGood(t *testing.T) {
	c := Config{}
	c.Output.Format = strp("json")
	c.Output.Color = strp("auto")
	c.Log.Level = strp("debug")
	c.Log.Format = strp("console")
	c.FloodWait.Mode = strp("wait")
	c.FloodWait.MaxSeconds = intp(0)
	require.NoError(t, ValidateEnums(c))
}

func TestValidateEnums_allNil_defaultsAreOK(t *testing.T) {
	require.NoError(t, ValidateEnums(Config{}))
}

func TestValidateEnums_badOutputFormat(t *testing.T) {
	c := Config{}
	c.Output.Format = strp("xml")
	err := ValidateEnums(c)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalid)
	require.Contains(t, err.Error(), "output.format")
}

func TestValidateEnums_badColor(t *testing.T) {
	c := Config{}
	c.Output.Color = strp("rainbow")
	err := ValidateEnums(c)
	require.ErrorIs(t, err, ErrInvalid)
}

func TestValidateEnums_badLogLevel(t *testing.T) {
	c := Config{}
	c.Log.Level = strp("trace")
	err := ValidateEnums(c)
	require.ErrorIs(t, err, ErrInvalid)
}

func TestValidateEnums_badLogFormat(t *testing.T) {
	c := Config{}
	c.Log.Format = strp("yaml")
	err := ValidateEnums(c)
	require.ErrorIs(t, err, ErrInvalid)
}

func TestValidateEnums_badFloodMode(t *testing.T) {
	c := Config{}
	c.FloodWait.Mode = strp("ignore")
	err := ValidateEnums(c)
	require.ErrorIs(t, err, ErrInvalid)
}

func TestValidateEnums_negativeFloodMax(t *testing.T) {
	c := Config{}
	c.FloodWait.MaxSeconds = intp(-1)
	err := ValidateEnums(c)
	require.ErrorIs(t, err, ErrInvalid)
}
