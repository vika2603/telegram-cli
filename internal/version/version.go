// Package version exposes the binary's version string. It is a leaf
// package (no imports) so every other internal package can read the
// value without introducing import cycles.
//
// Version is overridable at build time via:
//
//	go build -ldflags "-X github.com/vika2603/telegram-cli/internal/version.Version=<value>"
//
// Defaults to 0.0.0-dev so plain `go build` / `go install` still
// works. The Makefile's install / build targets stamp this with
// `0.0.0-${TAG}+${COMMIT}` automatically.
package version

// Version is the stamped build version. Do not modify at runtime.
var Version = "0.0.0-dev"
