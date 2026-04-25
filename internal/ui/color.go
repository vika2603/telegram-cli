package ui

import "os"

// ColorScheme styles strings for TTY output. When the owning IOStreams has
// color disabled, every method returns its argument unchanged.
type ColorScheme struct {
	enabled    bool
	support256 bool
	truecolor  bool
}

func newColorScheme(enabled, support256, truecolor bool) *ColorScheme {
	return &ColorScheme{enabled: enabled, support256: support256, truecolor: truecolor}
}

func (c *ColorScheme) wrap(code, s string) string {
	if !c.enabled {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func (c *ColorScheme) Red(s string) string       { return c.wrap("31", s) }
func (c *ColorScheme) Green(s string) string     { return c.wrap("32", s) }
func (c *ColorScheme) Yellow(s string) string    { return c.wrap("33", s) }
func (c *ColorScheme) Blue(s string) string      { return c.wrap("34", s) }
func (c *ColorScheme) Magenta(s string) string   { return c.wrap("35", s) }
func (c *ColorScheme) Cyan(s string) string      { return c.wrap("36", s) }
func (c *ColorScheme) Gray(s string) string      { return c.wrap("90", s) }
func (c *ColorScheme) Bold(s string) string      { return c.wrap("1", s) }
func (c *ColorScheme) Italic(s string) string    { return c.wrap("3", s) }
func (c *ColorScheme) Underline(s string) string { return c.wrap("4", s) }
func (c *ColorScheme) SuccessIcon() string       { return c.Green("✓") }
func (c *ColorScheme) FailureIcon() string       { return c.Red("✗") }

// ColorEnabled reports whether ColorScheme outputs ANSI codes.
func (s *IOStreams) ColorEnabled() bool { return s.colorEnabled }

// ColorSupport256 reports whether the terminal advertises 256-color support.
func (s *IOStreams) ColorSupport256() bool { return s.color256 }

// HasTrueColor reports whether the terminal advertises 24-bit color.
func (s *IOStreams) HasTrueColor() bool { return s.colorTruecolor }

// ColorScheme returns the attached scheme, rebuilt when the enabled flag
// has been toggled since last call.
func (s *IOStreams) ColorScheme() *ColorScheme {
	if s.scheme == nil || s.scheme.enabled != s.colorEnabled {
		s.scheme = newColorScheme(s.colorEnabled, s.color256, s.colorTruecolor)
	}
	return s.scheme
}

// SetColorEnabled overrides the auto-detected color state. Used by the
// root PersistentPreRunE when --color=always|never is set.
func (s *IOStreams) SetColorEnabled(v bool) {
	s.colorEnabled = v
	s.scheme = newColorScheme(v, s.color256, s.colorTruecolor)
}

// detectColor picks up environment hints: NO_COLOR disables; COLORTERM=
// truecolor/24bit enables truecolor; TERM containing "256color" enables
// 256; otherwise enable-if-stdout-is-tty.
func (s *IOStreams) detectColor() {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		s.colorEnabled = false
		return
	}
	if v := os.Getenv("COLORTERM"); v == "truecolor" || v == "24bit" {
		s.colorTruecolor = true
		s.color256 = true
		s.colorEnabled = s.stdoutTTY
		return
	}
	if term := os.Getenv("TERM"); containsSubstring(term, "256color") {
		s.color256 = true
	}
	s.colorEnabled = s.stdoutTTY
}

func containsSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
