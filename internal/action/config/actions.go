// Package config contains local configuration command actions.
package config

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"

	"github.com/pelletier/go-toml/v2"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/command"
	appconfig "github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// GetRequest is the normalized request for `tg config get`.
type GetRequest struct {
	Key          string
	ConfigPath   string
	NoRedact     bool
	ErrorIfUnset bool
}

// Get resolves and returns one config value with source annotation.
func Get(_ context.Context, req GetRequest) (output.ConfigKeyRow, error) {
	k, err := appconfig.ResolveKey(req.Key)
	if err != nil {
		return output.ConfigKeyRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	resolved, err := appconfig.LoadMergedWithSources(appconfig.Config{}, req.ConfigPath)
	if err != nil {
		return output.ConfigKeyRow{}, err
	}
	rawVal, present := appconfig.ReadRaw(resolved.Raw, k.Name)
	if !present && req.ErrorIfUnset {
		return output.ConfigKeyRow{}, fmt.Errorf("%w: key %q is not set", command.ErrPrecondition, k.Name)
	}
	return output.ConfigKeyRow{
		Key:    k.Name,
		Value:  typedValue(k, rawVal, present, req.NoRedact),
		Source: resolved.Sources[k.Name],
	}, nil
}

// SetRequest is the normalized request for `tg config set`.
type SetRequest struct {
	Key        string
	Value      string
	ConfigPath string
	Force      bool
	Yes        bool
	Prompter   ui.Prompter
}

// Set validates and writes one config value.
func Set(_ context.Context, req SetRequest) (output.ConfigKeyRow, error) {
	k, err := appconfig.ResolveKey(req.Key)
	if err != nil {
		return output.ConfigKeyRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	coerced, err := appconfig.CoerceValue(k, req.Value)
	if err != nil {
		return output.ConfigKeyRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if k.Name == "api_hash" && !req.Force {
		return output.ConfigKeyRow{}, fmt.Errorf("%w: setting api_hash requires --force (affects every account in this config)", command.ErrUsage)
	}
	if k.Name == "api_hash" {
		if err := ui.ConfirmDestructive(req.Prompter, "Overwrite shared api_hash?", req.Yes); err != nil {
			return output.ConfigKeyRow{}, err
		}
	}

	path := ResolveWritePath(req.ConfigPath)
	raw, err := appconfig.ReadRawAt(path)
	if err != nil {
		return output.ConfigKeyRow{}, err
	}
	oldRaw, _ := appconfig.ReadRaw(raw, k.Name)
	appconfig.WriteRaw(raw, k.Name, coerced)
	ensureVersion(raw)
	if err := appconfig.WriteRawAt(path, raw); err != nil {
		return output.ConfigKeyRow{}, err
	}
	return output.ConfigKeyRow{
		Action: "config_set",
		Key:    k.Name,
		Old:    displayValue(k, oldRaw),
		New:    displayValue(k, coerced),
	}, nil
}

// UnsetRequest is the normalized request for `tg config unset`.
type UnsetRequest struct {
	Key        string
	ConfigPath string
	Yes        bool
	Prompter   ui.Prompter
}

// Unset validates and removes one config value.
func Unset(_ context.Context, req UnsetRequest) (output.ConfigKeyRow, error) {
	k, err := appconfig.ResolveKey(req.Key)
	if err != nil {
		return output.ConfigKeyRow{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	if k.Name == "api_hash" {
		if err := ui.ConfirmDestructive(req.Prompter, "Unset shared api_hash?", req.Yes); err != nil {
			return output.ConfigKeyRow{}, err
		}
	}

	path := ResolveWritePath(req.ConfigPath)
	raw, err := appconfig.ReadRawAt(path)
	if err != nil {
		return output.ConfigKeyRow{}, err
	}
	oldRaw, present := appconfig.ReadRaw(raw, k.Name)
	if !present {
		return output.ConfigKeyRow{}, command.NewNoResultsError(fmt.Sprintf("key %q not set", k.Name))
	}
	appconfig.UnsetRaw(raw, k.Name)
	ensureVersion(raw)
	if err := appconfig.WriteRawAt(path, raw); err != nil {
		return output.ConfigKeyRow{}, err
	}
	return output.ConfigKeyRow{
		Action: "config_unset",
		Key:    k.Name,
		Old:    displayValue(k, oldRaw),
	}, nil
}

// ShowRequest is the normalized request for `tg config show`.
type ShowRequest struct {
	Config *appconfig.Config
}

// Show returns the redacted config and enriches it with default account state.
func Show(_ context.Context, req ShowRequest) (appconfig.RedactedConfig, error) {
	if req.Config == nil {
		return appconfig.RedactedConfig{}, fmt.Errorf("%w: config show called without config", command.ErrPrecondition)
	}
	r := appconfig.Redact(*req.Config)
	if req.Config.DefaultAccount != nil && *req.Config.DefaultAccount != "" {
		m, err := account.ReadMeta(*req.Config.DefaultAccount)
		if err != nil {
			return appconfig.RedactedConfig{}, fmt.Errorf("%w: default_account=%q but cannot read account meta: %w",
				command.ErrPrecondition, *req.Config.DefaultAccount, err)
		}
		state := string(m.State)
		r.AccountState = &state
	}
	return r, nil
}

// PathRequest is the normalized request for `tg config path`.
type PathRequest struct {
	FlagPath string
}

// Path returns the config path after applying CLI flag and default path rules.
func Path(_ context.Context, req PathRequest) string {
	return appconfig.ResolvePath(req.FlagPath, account.ConfigFile())
}

// PathResult is the JSON shape for `tg config path`.
type PathResult struct {
	Path string `json:"path"`
}

// EditRequest is the normalized request for `tg config edit`.
type EditRequest struct {
	ConfigPath    string
	IOStreams     *ui.IOStreams
	Prompter      ui.Prompter
	ResolveEditor func() ([]string, error)
}

// Edit opens an editor on the config file, validates it, and atomically
// replaces the live config on success.
func Edit(ctx context.Context, req EditRequest) (map[string]any, error) {
	_ = ctx
	if req.ResolveEditor == nil {
		return nil, fmt.Errorf("%w: internal error: config editor resolver is not configured", command.ErrPrecondition)
	}
	editorCmd, err := req.ResolveEditor()
	if err != nil {
		return nil, err
	}

	path := ResolveWritePath(req.ConfigPath)
	initialRaw, _ := appconfig.ReadRawAt(path)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(path), ".config-edit-*.toml")
	if err != nil {
		return nil, err
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, statErr := os.Stat(path); statErr == nil {
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			_ = tmpFile.Close()
			return nil, rerr
		}
		if _, werr := tmpFile.Write(b); werr != nil {
			_ = tmpFile.Close()
			return nil, werr
		}
	} else {
		_, _ = tmpFile.WriteString("# tg config\nversion = 1\n")
	}
	if err := tmpFile.Chmod(0600); err != nil {
		_ = tmpFile.Close()
		return nil, err
	}
	if err := tmpFile.Close(); err != nil {
		return nil, err
	}

	for {
		argv := append(editorCmd, tmpPath) //nolint:gocritic // intentional append-then-use in loop
		c := exec.Command(argv[0], argv[1:]...)
		c.Stdin = req.IOStreams.In
		c.Stdout = req.IOStreams.ErrOut
		c.Stderr = req.IOStreams.ErrOut
		if runErr := c.Run(); runErr != nil {
			return nil, fmt.Errorf("%w: editor exited non-zero: %s", command.ErrUsage, runErr.Error())
		}

		var warnings []string
		cfg, loadErr := appconfig.Load(tmpPath, func(msg string) { warnings = append(warnings, msg) })
		if loadErr != nil {
			_, _ = fmt.Fprintf(req.IOStreams.ErrOut, "parse error: %s\n", loadErr.Error())
			if !promptReopen(req.Prompter) {
				return nil, command.ErrNotConfirmed
			}
			continue
		}

		if enumErr := appconfig.ValidateEnums(cfg); enumErr != nil {
			_, _ = fmt.Fprintf(req.IOStreams.ErrOut, "validation error: %s\n", enumErr.Error())
			if !promptReopen(req.Prompter) {
				return nil, command.ErrNotConfirmed
			}
			continue
		}

		for _, w := range warnings {
			_, _ = fmt.Fprintf(req.IOStreams.ErrOut, "warning: %s\n", w)
		}

		newBytes, readErr := os.ReadFile(tmpPath)
		if readErr != nil {
			return nil, readErr
		}
		var parsed map[string]any
		_ = toml.Unmarshal(newBytes, &parsed)

		if renameErr := os.Rename(tmpPath, path); renameErr != nil {
			return nil, renameErr
		}
		return map[string]any{"action": "config_edit", "changes": diffCount(initialRaw, parsed)}, nil
	}
}

// ResolveWritePath chooses the target config path for commands that write.
func ResolveWritePath(flagPath string) string {
	if flagPath != "" {
		return flagPath
	}
	return account.ConfigFile()
}

// HumanValue is the plain-text representation for config get output.
func HumanValue(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// DefaultResolveEditor returns the editor argv.
func DefaultResolveEditor() ([]string, error) {
	for _, env := range []string{"VISUAL", "EDITOR"} {
		if v := os.Getenv(env); v != "" {
			return []string{"/bin/sh", "-c", v + ` "$1"`, "editor"}, nil
		}
	}
	for _, cand := range []string{"vim", "vi"} {
		if p, err := exec.LookPath(cand); err == nil {
			return []string{p}, nil
		}
	}
	return nil, fmt.Errorf("%w: no editor found; set $VISUAL or $EDITOR", command.ErrPrecondition)
}

func typedValue(k appconfig.Key, raw any, present, noRedact bool) any {
	if !present {
		return nil
	}
	if k.Name == "api_hash" && !noRedact {
		return displayValue(k, raw)
	}
	return raw
}

func displayValue(k appconfig.Key, raw any) any {
	if k.Name == "api_hash" {
		if s, ok := raw.(string); ok && s != "" {
			return appconfig.MaskAPIHash(s)
		}
		if raw != nil {
			return "****"
		}
	}
	return raw
}

func ensureVersion(raw map[string]any) {
	if _, ok := raw["version"]; !ok {
		raw["version"] = int64(1)
	}
}

func promptReopen(prompter ui.Prompter) bool {
	ok, _ := prompter.Confirm("Re-open editor?", true)
	return ok
}

func diffCount(a, b map[string]any) int {
	keys := map[string]struct{}{}
	for k := range a {
		keys[k] = struct{}{}
	}
	for k := range b {
		keys[k] = struct{}{}
	}
	n := 0
	for k := range keys {
		if !reflect.DeepEqual(a[k], b[k]) {
			n++
		}
	}
	return n
}
