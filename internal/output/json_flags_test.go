package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

type testItem struct {
	Name  string `json:"name"`
	State string `json:"state"`
	APIID int    `json:"api_id"`
}

func TestAddJSONFlags_NoFlag_ExporterNil(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	var exp output.Exporter
	output.AddJSONFlags(cmd, &exp, []string{"name", "state", "api_id"})
	cmd.RunE = func(*cobra.Command, []string) error { return nil }
	require.NoError(t, cmd.Execute())
	require.Nil(t, exp, "--json not set => exporter stays nil")
}

func TestAddJSONFlags_JSONFlagBuildsExporter(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	cmd := &cobra.Command{Use: "x"}
	var exp output.Exporter
	output.AddJSONFlags(cmd, &exp, []string{"name", "state", "api_id"})
	cmd.SetArgs([]string{"--json=name,state"})
	cmd.RunE = func(*cobra.Command, []string) error { return nil }
	require.NoError(t, cmd.Execute())
	require.NotNil(t, exp)
	require.ElementsMatch(t, []string{"name", "state"}, exp.Fields())

	item := testItem{Name: "work", State: "AUTHED", APIID: 42}
	require.NoError(t, exp.Write(ios, item))
	require.Contains(t, stdout.String(), `"name":"work"`)
	require.Contains(t, stdout.String(), `"state":"AUTHED"`)
	require.NotContains(t, stdout.String(), `"api_id"`)
}

func TestAddJSONFlags_UnknownFieldFails(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	var exp output.Exporter
	output.AddJSONFlags(cmd, &exp, []string{"name"})
	cmd.SetArgs([]string{"--json=wrong_field"})
	cmd.RunE = func(*cobra.Command, []string) error { return nil }
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "wrong_field")
}

func TestAddJSONFlags_JQWithoutJSONFails(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	var exp output.Exporter
	output.AddJSONFlags(cmd, &exp, []string{"name"})
	cmd.SetArgs([]string{"--jq", ".name"})
	cmd.RunE = func(*cobra.Command, []string) error { return nil }
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "--jq")
}

func TestAddJSONFlags_JQFiltersOutput(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	cmd := &cobra.Command{Use: "x"}
	var exp output.Exporter
	output.AddJSONFlags(cmd, &exp, []string{"name", "state"})
	cmd.SetArgs([]string{"--json=name,state", "--jq", ".name"})
	cmd.RunE = func(*cobra.Command, []string) error { return nil }
	require.NoError(t, cmd.Execute())
	require.NotNil(t, exp)
	require.NoError(t, exp.Write(ios, testItem{Name: "work", State: "AUTHED"}))
	require.Contains(t, stdout.String(), `"work"`)
	require.NotContains(t, stdout.String(), `"AUTHED"`)
}

func TestAddJSONFlags_TemplateRenders(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	cmd := &cobra.Command{Use: "x"}
	var exp output.Exporter
	output.AddJSONFlags(cmd, &exp, []string{"name"})
	cmd.SetArgs([]string{"--json=name", "--template", "{{.name}}-done"})
	cmd.RunE = func(*cobra.Command, []string) error { return nil }
	require.NoError(t, cmd.Execute())
	require.NoError(t, exp.Write(ios, testItem{Name: "work"}))
	require.Equal(t, "work-done\n", stdout.String())
}

func TestAddJSONFlags_ListEmitsNDJSON(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	cmd := &cobra.Command{Use: "x"}
	var exp output.Exporter
	output.AddJSONFlags(cmd, &exp, []string{"name", "state"})
	cmd.SetArgs([]string{"--json=name,state"})
	cmd.RunE = func(*cobra.Command, []string) error { return nil }
	require.NoError(t, cmd.Execute())

	items := []testItem{
		{Name: "work", State: "AUTHED"},
		{Name: "personal", State: "NEW"},
	}
	require.NoError(t, exp.Write(ios, items))

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	require.Len(t, lines, 2, "list should emit one line per item, got %q", stdout.String())
	require.NotContains(t, stdout.String(), "[", "output must not contain a JSON array wrapper")
	for _, line := range lines {
		require.True(t, strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}"),
			"each line should be a JSON object, got %q", line)
	}
	require.Contains(t, stdout.String(), `"name":"work"`)
	require.Contains(t, stdout.String(), `"name":"personal"`)
}

func TestAddJSONFlags_EmptyListEmitsNothing(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	cmd := &cobra.Command{Use: "x"}
	var exp output.Exporter
	output.AddJSONFlags(cmd, &exp, []string{"name"})
	cmd.SetArgs([]string{"--json=name"})
	cmd.RunE = func(*cobra.Command, []string) error { return nil }
	require.NoError(t, cmd.Execute())

	require.NoError(t, exp.Write(ios, []testItem{}))
	require.Empty(t, stdout.String(), "empty list should emit nothing, got %q", stdout.String())
}

func TestAddJSONFlags_NilSliceEmitsNothing(t *testing.T) {
	// A typed nil slice (the zero value of `var rows []T`) must render as
	// ndjson with zero lines, not the literal "null". This is what most
	// Fetch closures return when the upstream iterator yielded nothing.
	ios, _, stdout, _ := ui.Test()
	cmd := &cobra.Command{Use: "x"}
	var exp output.Exporter
	output.AddJSONFlags(cmd, &exp, []string{"name"})
	cmd.SetArgs([]string{"--json=name"})
	cmd.RunE = func(*cobra.Command, []string) error { return nil }
	require.NoError(t, cmd.Execute())

	var rows []testItem
	require.NoError(t, exp.Write(ios, rows))
	require.Empty(t, stdout.String(), "nil slice should emit nothing, got %q", stdout.String())
}

func TestAddJSONFlags_JQAppliedPerItem(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	cmd := &cobra.Command{Use: "x"}
	var exp output.Exporter
	output.AddJSONFlags(cmd, &exp, []string{"name"})
	cmd.SetArgs([]string{"--json=name", "--jq", ".name"})
	cmd.RunE = func(*cobra.Command, []string) error { return nil }
	require.NoError(t, cmd.Execute())

	items := []testItem{
		{Name: "work"},
		{Name: "personal"},
	}
	require.NoError(t, exp.Write(ios, items))

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	require.Equal(t, []string{`"work"`, `"personal"`}, lines)
}

func TestAddJSONFlags_TemplateAppliedPerItem(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	cmd := &cobra.Command{Use: "x"}
	var exp output.Exporter
	output.AddJSONFlags(cmd, &exp, []string{"name"})
	cmd.SetArgs([]string{"--json=name", "--template", "{{.name}}-done"})
	cmd.RunE = func(*cobra.Command, []string) error { return nil }
	require.NoError(t, cmd.Execute())

	items := []testItem{
		{Name: "work"},
		{Name: "personal"},
	}
	require.NoError(t, exp.Write(ios, items))
	require.Equal(t, "work-done\npersonal-done\n", stdout.String())
}

func TestAddJSONFlags_ChainsExistingPreRunE(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	var priorCalled bool
	cmd.PreRunE = func(*cobra.Command, []string) error { priorCalled = true; return nil }
	var exp output.Exporter
	output.AddJSONFlags(cmd, &exp, []string{"name"})
	cmd.RunE = func(*cobra.Command, []string) error { return nil }
	require.NoError(t, cmd.Execute())
	require.True(t, priorCalled, "prior PreRunE must still run")
}
