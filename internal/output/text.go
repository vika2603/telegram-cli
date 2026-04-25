package output

import (
	"strings"

	xansi "github.com/charmbracelet/x/ansi"
)

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func fitText(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	return xansi.Truncate(s, limit, "…")
}

func padRight(s string, width int) string {
	padding := width - displayWidth(s)
	if padding <= 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
}

func displayWidth(s string) int {
	return xansi.StringWidth(s)
}
