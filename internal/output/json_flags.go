package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Exporter renders typed data as JSON / jq-filtered / template-rendered
// output. AddJSONFlags wires --json/--jq/--template onto a cobra.Command
// and populates *exporter when those flags are present.
type Exporter interface {
	Fields() []string
	Write(io *ui.IOStreams, data any) error
}

// AddJSONFlags registers --json, --jq, --template onto cmd and installs a
// PreRunE that validates their combination. If cmd already has a PreRunE,
// the prior one is chained: prior first, then json validation.
func AddJSONFlags(cmd *cobra.Command, exporter *Exporter, fields []string) {
	var (
		jsonFields string
		jqExpr     string
		tmplExpr   string
	)
	cmd.Flags().StringVar(&jsonFields, "json", "", "emit JSON; value is comma-separated field subset (omit value for all fields)")
	cmd.Flag("json").NoOptDefVal = "*"
	cmd.Flags().StringVarP(&jqExpr, "jq", "q", "", "filter JSON output through a jq expression")
	cmd.Flags().StringVarP(&tmplExpr, "template", "t", "", "format JSON output using a Go template")

	_ = cmd.RegisterFlagCompletionFunc("json", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return fields, cobra.ShellCompDirectiveNoFileComp
	})

	prior := cmd.PreRunE
	cmd.PreRunE = func(c *cobra.Command, args []string) error {
		if prior != nil {
			if err := prior(c, args); err != nil {
				return err
			}
		}
		if !c.Flag("json").Changed {
			if jqExpr != "" {
				return flagErrorf("--jq requires --json")
			}
			if tmplExpr != "" {
				return flagErrorf("--template requires --json")
			}
			return nil
		}
		wanted := fields
		if jsonFields != "*" && jsonFields != "" {
			parts := strings.Split(jsonFields, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				found := false
				for _, f := range fields {
					if f == p {
						found = true
						break
					}
				}
				if !found {
					return flagErrorf("unknown JSON field %q (valid: %s)", p, strings.Join(fields, ", "))
				}
			}
			wanted = parseCSV(jsonFields)
		}
		je := &jsonExporter{fields: wanted}
		if jqExpr != "" {
			q, err := gojq.Parse(jqExpr)
			if err != nil {
				return flagErrorf("--jq parse error: %v", err)
			}
			code, err := gojq.Compile(q)
			if err != nil {
				return flagErrorf("--jq compile error: %v", err)
			}
			je.jq = code
		}
		if tmplExpr != "" {
			tmpl, err := template.New("tg").Parse(tmplExpr)
			if err != nil {
				return flagErrorf("--template parse error: %v", err)
			}
			je.tmpl = tmpl
		}
		*exporter = je
		return nil
	}
}

type jsonExporter struct {
	fields []string
	jq     *gojq.Code
	tmpl   *template.Template
}

func (e *jsonExporter) Fields() []string { return append([]string(nil), e.fields...) }

func (e *jsonExporter) Write(io *ui.IOStreams, data any) error {
	normalized, err := normalize(data, e.fields)
	if err != nil {
		return err
	}
	// List shape -> ndjson (one line per element, empty slice = no output).
	// Any other shape -> single line.
	if items, ok := normalized.([]any); ok {
		for _, item := range items {
			if err := e.renderOne(io, item); err != nil {
				return err
			}
		}
		return nil
	}
	return e.renderOne(io, normalized)
}

// renderOne writes exactly one item to io.Out, terminated by a newline,
// honoring the active render mode (jq > template > plain JSON). For jq,
// one input may yield multiple output values; each goes on its own
// line, which matches standard jq semantics.
func (e *jsonExporter) renderOne(io *ui.IOStreams, item any) error {
	if e.jq != nil {
		iter := e.jq.Run(item)
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, isErr := v.(error); isErr {
				return err
			}
			b, err := json.Marshal(v)
			if err != nil {
				return err
			}
			_, _ = io.Out.Write(append(b, '\n'))
		}
		return nil
	}
	if e.tmpl != nil {
		var buf bytes.Buffer
		if err := e.tmpl.Execute(&buf, item); err != nil {
			return err
		}
		_, _ = io.Out.Write(append(buf.Bytes(), '\n'))
		return nil
	}
	b, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, _ = io.Out.Write(append(b, '\n'))
	return nil
}

// normalize marshals data through JSON so concrete Go types collapse
// into gojq-native shapes: map[string]any, []any, primitives, or nil.
// It then applies the field filter: for a map, drop keys not in
// fields; for a list of maps, drop keys per element. Primitives and
// non-map lists pass through unfiltered. fields empty means keep all.
func normalize(data any, fields []string) (any, error) {
	// A typed nil slice marshals to "null"; ensure it renders as an empty
	// ndjson stream (zero lines) instead of the literal "null".
	if data != nil {
		rv := reflect.ValueOf(data)
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return []any{}, nil
		}
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var root any
	if err := json.Unmarshal(b, &root); err != nil {
		return nil, err
	}
	switch v := root.(type) {
	case map[string]any:
		return filterMap(v, fields), nil
	case []any:
		out := make([]any, 0, len(v))
		for _, it := range v {
			if m, ok := it.(map[string]any); ok {
				out = append(out, filterMap(m, fields))
			} else {
				out = append(out, it)
			}
		}
		return out, nil
	default:
		return root, nil
	}
}

func filterMap(m map[string]any, fields []string) map[string]any {
	if len(fields) == 0 {
		return m
	}
	want := map[string]struct{}{}
	for _, f := range fields {
		want[f] = struct{}{}
	}
	out := map[string]any{}
	for k, v := range m {
		if _, ok := want[k]; ok {
			out[k] = v
		}
	}
	return out
}

func parseCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func flagErrorf(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{command.ErrUsage}, a...)...)
}
