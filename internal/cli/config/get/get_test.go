package get_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/config/get"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// newTestRoot creates a minimal parent command that exposes the persistent
// flags internal/cli/root registers, so subcommands can access them in tests.
func newTestRoot(sub *cobra.Command) *cobra.Command {
	root := &cobra.Command{Use: "tg", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().String("config", "", "Config file path")
	root.AddCommand(sub)
	return root
}

// clearTGEnv unsets environment variables that FromEnv reads so that tests
// using LoadMergedWithSources start from a clean slate.
func clearTGEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"TG_ACCOUNT", "TG_API_ID", "TG_API_HASH", "TG_OUTPUT", "TG_COLOR",
		"TG_LOG_LEVEL", "TG_LOG_FORMAT", "TG_FLOOD_WAIT_MODE",
		"TG_FLOOD_WAIT_MAX", "TG_CONFIG",
	} {
		t.Setenv(k, "")
	}
	// TG_LOG_FILE must be fully unset (not set to "") because FromEnv treats
	// the empty-string case as a meaningful override.
	orig, hadOrig := os.LookupEnv("TG_LOG_FILE")
	if err := os.Unsetenv("TG_LOG_FILE"); err != nil {
		t.Fatalf("clearTGEnv unsetenv TG_LOG_FILE: %v", err)
	}
	t.Cleanup(func() {
		if hadOrig {
			_ = os.Setenv("TG_LOG_FILE", orig)
		} else {
			_ = os.Unsetenv("TG_LOG_FILE")
		}
	})
}

// writeTempConfig writes a TOML config file in a temp dir and returns its path.
func writeTempConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(p, []byte(body), 0600))
	return p
}

func TestNew_ParsesFlags(t *testing.T) {
	var captured *get.Options
	f := runtime.NewTestInvocation(t)
	cmd := get.New(f, func(o *get.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs(strings.Fields("output.format"))
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.Equal(t, "output.format", captured.Key)
	require.False(t, captured.NoRedact)
	require.False(t, captured.ErrorIfUnset)
}

func TestNew_ParsesNoRedact(t *testing.T) {
	var captured *get.Options
	f := runtime.NewTestInvocation(t)
	cmd := get.New(f, func(o *get.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs(strings.Fields("api_hash --no-redact"))
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.True(t, captured.NoRedact)
}

func TestNew_ParsesErrorIfUnset(t *testing.T) {
	var captured *get.Options
	f := runtime.NewTestInvocation(t)
	cmd := get.New(f, func(o *get.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs(strings.Fields("default_account --error-if-unset"))
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.True(t, captured.ErrorIfUnset)
}

func TestRun_UnknownKey_ErrUsage(t *testing.T) {
	clearTGEnv(t)
	configPath := filepath.Join(t.TempDir(), "nonexistent.toml")

	ios, _, _, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.ConfigPath = configPath

	cmd := get.New(f, nil)
	cmd.SetArgs([]string{"not_a_real_key"})
	err := cmd.Execute()
	require.Error(t, err)
	require.ErrorIs(t, err, command.ErrUsage)
	require.Contains(t, err.Error(), "valid keys")
}

func TestRun_UnsetKey_EmptyLine(t *testing.T) {
	clearTGEnv(t)
	configPath := filepath.Join(t.TempDir(), "nonexistent.toml")

	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.ConfigPath = configPath

	cmd := get.New(f, nil)
	cmd.SetArgs([]string{"default_account"})
	require.NoError(t, cmd.Execute())
	// Unset key prints empty line.
	require.Equal(t, "\n", stdout.String())
}

func TestRun_UnsetKeyErrorIfUnset_ErrPrecondition(t *testing.T) {
	clearTGEnv(t)
	configPath := filepath.Join(t.TempDir(), "nonexistent.toml")

	ios, _, _, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.ConfigPath = configPath

	cmd := get.New(f, nil)
	cmd.SetArgs(strings.Fields("default_account --error-if-unset"))
	err := cmd.Execute()
	require.Error(t, err)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_SetKey_PrintsValue(t *testing.T) {
	clearTGEnv(t)
	configPath := writeTempConfig(t, `version = 1
default_account = "alice"`)

	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.ConfigPath = configPath

	cmd := get.New(f, nil)
	cmd.SetArgs([]string{"default_account"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "alice\n", stdout.String())
}

func TestRun_DefaultKey_PrintsValue(t *testing.T) {
	clearTGEnv(t)
	configPath := filepath.Join(t.TempDir(), "nonexistent.toml")

	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.ConfigPath = configPath

	cmd := get.New(f, nil)
	cmd.SetArgs([]string{"output.format"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "human\n", stdout.String())
}

func TestRun_APIHash_DefaultMasked(t *testing.T) {
	clearTGEnv(t)
	configPath := writeTempConfig(t, `version = 1
api_hash = "abcdefghijklmnopqrstuvwxyz"`)

	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.ConfigPath = configPath

	cmd := get.New(f, nil)
	cmd.SetArgs([]string{"api_hash"})
	require.NoError(t, cmd.Execute())
	out := stdout.String()
	require.NotContains(t, out, "abcdefghijklmnopqrstuvwxyz")
	require.Contains(t, out, "****")
}

func TestRun_APIHash_NoRedact(t *testing.T) {
	clearTGEnv(t)
	configPath := writeTempConfig(t, `version = 1
api_hash = "abcdefghijklmnopqrstuvwxyz"`)

	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.ConfigPath = configPath

	cmd := get.New(f, nil)
	cmd.SetArgs(strings.Fields("api_hash --no-redact"))
	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), "abcdefghijklmnopqrstuvwxyz")
}

func TestRun_JSONOutput_HasKeyValueSource(t *testing.T) {
	clearTGEnv(t)
	configPath := filepath.Join(t.TempDir(), "nonexistent.toml")

	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.ConfigPath = configPath

	sub := get.New(f, nil)
	root := newTestRoot(sub)
	root.SetArgs(strings.Fields("get output.format --json"))
	require.NoError(t, root.Execute())

	out := strings.TrimSpace(stdout.String())
	require.NotEmpty(t, out)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	require.Contains(t, result, "key")
	require.Contains(t, result, "value")
	require.Contains(t, result, "source")
	require.Equal(t, "output.format", result["key"])
	require.Equal(t, "human", result["value"])
	require.Equal(t, "default", result["source"])
}

// TestRun_TypedValue_APIID_IsNumericInJSON verifies that api_id is a JSON
// number (float64 when decoded as any) rather than a stringified integer.
func TestRun_TypedValue_APIID_IsNumericInJSON(t *testing.T) {
	clearTGEnv(t)
	configPath := writeTempConfig(t, `version = 1
api_id = 12345`)

	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.ConfigPath = configPath

	sub := get.New(f, nil)
	root := newTestRoot(sub)
	root.SetArgs(strings.Fields("get api_id --json"))
	require.NoError(t, root.Execute())

	out := strings.TrimSpace(stdout.String())
	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	// Go's json.Unmarshal into any decodes JSON numbers as float64. Assert
	// the dynamic type is float64 (not string) to confirm api_id is a JSON
	// number, then check the integer value.
	v, ok := result["value"].(float64)
	require.True(t, ok, "api_id must be a JSON number, not a string; got %T", result["value"])
	require.InEpsilon(t, float64(12345), v, 0, "api_id value must be 12345")
}

// TestRun_TypedValue_OutputFormat_IsStringInJSON verifies that an enum key
// value is a JSON string.
func TestRun_TypedValue_OutputFormat_IsStringInJSON(t *testing.T) {
	clearTGEnv(t)
	configPath := writeTempConfig(t, `version = 1
[output]
format = "json"`)

	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.ConfigPath = configPath

	sub := get.New(f, nil)
	root := newTestRoot(sub)
	root.SetArgs(strings.Fields("get output.format --json"))
	require.NoError(t, root.Execute())

	out := strings.TrimSpace(stdout.String())
	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	require.Equal(t, "json", result["value"])
}

// TestRun_Source_DefaultAccount_NeverFlag checks that account selection via
// --account does not produce a "flag" source for default_account. The invocation
// pattern used here mirrors how root.go wires things: the ConfigPath field
// carries only the --config flag value, not the --account value.
func TestRun_Source_DefaultAccount_NeverFlag(t *testing.T) {
	clearTGEnv(t)
	configPath := writeTempConfig(t, `version = 1
default_account = "alice"`)

	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.ConfigPath = configPath
	// Simulate --account being passed: AccountName is set on the invocation but
	// is NOT wired into the config precedence pipeline.
	f.AccountName = "work"

	sub := get.New(f, nil)
	root := newTestRoot(sub)
	root.SetArgs(strings.Fields("get default_account --json"))
	require.NoError(t, root.Execute())

	out := strings.TrimSpace(stdout.String())
	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	// Value comes from the file (which set default_account = "alice"), not
	// from the --account flag (which is a runtime selector, not config layer).
	require.Equal(t, "file", result["source"])
	require.Equal(t, "alice", result["value"])
	_ = stdout
}

// TestRun_Alias_FromFile reads an alias key set in the config file.
func TestRun_Alias_FromFile(t *testing.T) {
	clearTGEnv(t)
	configPath := writeTempConfig(t, `version = 1

[aliases]
boss = "@ceo"`)

	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.ConfigPath = configPath

	cmd := get.New(f, nil)
	cmd.SetArgs([]string{"aliases.boss"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "@ceo\n", stdout.String())
}
