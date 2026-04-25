package output

import (
	"fmt"
	"strings"

	"github.com/vika2603/telegram-cli/internal/ui"
)

// TablePrinter renders plain command output. It intentionally stays small:
// our list views are domain-shaped summaries, not a generic table product.
// Width math is delegated to Charm's ANSI helpers in text.go.
type TablePrinter struct {
	io     *ui.IOStreams
	header []string
	rows   [][]string
	hasHdr bool
}

// NewTablePrinter returns a fresh printer bound to the given IOStreams.
func NewTablePrinter(io *ui.IOStreams) *TablePrinter {
	return &TablePrinter{io: io}
}

// AddHeader sets the column headers. Calling twice overwrites.
func (t *TablePrinter) AddHeader(cells ...string) {
	t.header = cells
	t.hasHdr = true
}

// AddRow appends a row. Fewer cells than the header pads the trailing columns;
// more cells extends the column count.
func (t *TablePrinter) AddRow(cells ...string) {
	t.rows = append(t.rows, cells)
}

// Render writes the table. TTY output is aligned; non-TTY output is TSV with
// headers so scripts retain stable column names.
func (t *TablePrinter) Render() error {
	if !t.io.IsStdoutTTY() {
		return t.renderTabSeparated()
	}
	return t.renderAligned()
}

func (t *TablePrinter) renderTabSeparated() error {
	if t.hasHdr {
		if _, err := fmt.Fprintln(t.io.Out, strings.Join(t.header, "\t")); err != nil {
			return err
		}
	}
	for _, r := range t.rows {
		if _, err := fmt.Fprintln(t.io.Out, strings.Join(r, "\t")); err != nil {
			return err
		}
	}
	return nil
}

func (t *TablePrinter) renderAligned() error {
	cols := t.countCols()
	widths := make([]int, cols)
	if t.hasHdr {
		for i, h := range t.header {
			if w := displayWidth(h); i < cols && w > widths[i] {
				widths[i] = w
			}
		}
	}
	for _, r := range t.rows {
		for i, v := range r {
			if w := displayWidth(v); i < cols && w > widths[i] {
				widths[i] = w
			}
		}
	}
	if t.hasHdr {
		if err := t.writeRow(t.header, widths, cols); err != nil {
			return err
		}
	}
	for _, r := range t.rows {
		if err := t.writeRow(r, widths, cols); err != nil {
			return err
		}
	}
	return nil
}

func (t *TablePrinter) countCols() int {
	n := len(t.header)
	for _, r := range t.rows {
		if len(r) > n {
			n = len(r)
		}
	}
	return n
}

func (t *TablePrinter) writeRow(r []string, widths []int, cols int) error {
	var b strings.Builder
	for i := range cols {
		v := ""
		if i < len(r) {
			v = r[i]
		}
		if i < cols-1 {
			b.WriteString(padRight(v, widths[i]))
			b.WriteString("  ")
		} else {
			b.WriteString(v)
		}
	}
	_, err := fmt.Fprintln(t.io.Out, b.String())
	return err
}
