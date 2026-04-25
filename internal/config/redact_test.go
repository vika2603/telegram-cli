package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedact_masksAPIHash(t *testing.T) {
	h := "abcd1234secretrest"
	c := Config{APIHash: &h}
	r := Redact(c)
	require.NotNil(t, r.APIHash)
	require.Equal(t, "abcd1234…****", *r.APIHash)
}

func TestRedact_shortAPIHashFullyMasked(t *testing.T) {
	h := "short"
	c := Config{APIHash: &h}
	r := Redact(c)
	require.Equal(t, "****", *r.APIHash)
}

func TestRedact_noAPIHashFieldStaysNil(t *testing.T) {
	c := Config{}
	r := Redact(c)
	require.Nil(t, r.APIHash)
}

func TestRedact_copiesOtherFieldsVerbatim(t *testing.T) {
	acct := "alice"
	fmt := "json"
	c := Config{DefaultAccount: &acct, Output: OutputCfg{Format: &fmt}}
	r := Redact(c)
	require.Equal(t, "alice", *r.DefaultAccount)
	require.Equal(t, "json", *r.Output.Format)
}

func TestRedactedConfig_humanMentionsMaskedHash(t *testing.T) {
	h := "abcd1234secretrest"
	c := Config{Version: ptr(1), APIHash: &h}
	s := Redact(c).Human()
	require.Contains(t, s, "abcd1234…****")
	require.NotContains(t, s, "secretrest")
}
