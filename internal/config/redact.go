package config

import (
	"fmt"
	"strconv"
)

// RedactedConfig is the JSON shape emitted by `tg config show`. Same fields
// as Config plus `account_state`, with api_hash in its masked form.
// AccountState is populated by the caller (the config package does not
// import account), so Redact leaves it nil.
type RedactedConfig struct {
	Version        *int              `json:"version,omitempty"`
	DefaultAccount *string           `json:"default_account,omitempty"`
	AccountState   *string           `json:"account_state,omitempty"`
	APIID          *int              `json:"api_id,omitempty"`
	APIHash        *string           `json:"api_hash,omitempty"`
	Output         OutputCfg         `json:"output"`
	Log            LogCfg            `json:"log"`
	FloodWait      FloodWaitCfg      `json:"flood_wait"`
	Aliases        map[string]string `json:"aliases,omitempty"`
}

// Redact returns a RedactedConfig with api_hash masked. First 8 chars are
// preserved so users can distinguish environments; the remainder becomes
// "…****". For api_hash of 8 chars or fewer the whole value becomes "****".
func Redact(c Config) RedactedConfig {
	out := RedactedConfig{
		Version:        c.Version,
		DefaultAccount: c.DefaultAccount,
		APIID:          c.APIID,
		Output:         c.Output,
		Log:            c.Log,
		FloodWait:      c.FloodWait,
		Aliases:        c.Aliases,
	}
	if c.APIHash != nil {
		m := maskAPIHash(*c.APIHash)
		out.APIHash = &m
	}
	return out
}

func maskAPIHash(h string) string {
	if len(h) <= 8 {
		return "****"
	}
	return h[:8] + "…****"
}

// Human returns a grep-friendly `key = value` dump with [section] headers.
// Shape is stable within v1 but the JSON form is the machine contract.
func (r RedactedConfig) Human() string {
	var b []byte
	add := func(k, v string) { b = append(b, []byte(k+" = "+v+"\n")...) }
	if r.Version != nil {
		add("version", strconv.Itoa(*r.Version))
	}
	if r.DefaultAccount != nil {
		add("default_account", fmt.Sprintf("%q", *r.DefaultAccount))
	}
	if r.AccountState != nil {
		add("account_state", fmt.Sprintf("%q", *r.AccountState))
	}
	if r.APIID != nil {
		add("api_id", strconv.Itoa(*r.APIID))
	}
	if r.APIHash != nil {
		add("api_hash", fmt.Sprintf("%q", *r.APIHash))
	}
	b = append(b, []byte("\n[output]\n")...)
	if r.Output.Format != nil {
		add("format", fmt.Sprintf("%q", *r.Output.Format))
	}
	if r.Output.Color != nil {
		add("color", fmt.Sprintf("%q", *r.Output.Color))
	}
	b = append(b, []byte("\n[log]\n")...)
	if r.Log.Level != nil {
		add("level", fmt.Sprintf("%q", *r.Log.Level))
	}
	if r.Log.File != nil {
		add("file", fmt.Sprintf("%q", *r.Log.File))
	}
	if r.Log.Format != nil {
		add("format", fmt.Sprintf("%q", *r.Log.Format))
	}
	b = append(b, []byte("\n[flood_wait]\n")...)
	if r.FloodWait.Mode != nil {
		add("mode", fmt.Sprintf("%q", *r.FloodWait.Mode))
	}
	if r.FloodWait.MaxSeconds != nil {
		add("max_seconds", strconv.Itoa(*r.FloodWait.MaxSeconds))
	}
	if len(r.Aliases) > 0 {
		b = append(b, []byte("\n[aliases]\n")...)
		for k, v := range r.Aliases {
			add(k, fmt.Sprintf("%q", v))
		}
	}
	return string(b)
}
