// Package program is the real main entry. cmd/tg/main.go is a thin shim
// that just delegates here so the Main() function is testable and the
// wire-up can import internal/ packages (which cmd/ technically can).
package program

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/cli/root"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/runtime/defaults"
	"github.com/vika2603/telegram-cli/internal/version"
)

// Main constructs the Invocation + root cobra tree, dispatches to cobra, and
// returns the exit code mapped through output.EmitError. It never calls
// os.Exit itself — the shim in cmd/tg/main.go owns that.
func Main() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	f := defaults.New(version.Version)
	rootCmd := root.New(f)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		mode := resolveErrorMode(resolveExecutedCmd(rootCmd), f)
		return output.EmitError(f.IOStreams.ErrOut, mode, err)
	}
	return 0
}

// resolveExecutedCmd finds the leaf cobra command that actually ran, so
// resolveErrorMode sees per-command flags (like --json) and not just the
// root. Falls back to the root on parse error or empty argv.
func resolveExecutedCmd(rootCmd *cobra.Command) *cobra.Command {
	if len(os.Args) < 2 {
		return rootCmd
	}
	cmd, _, err := rootCmd.Find(os.Args[1:])
	if err != nil || cmd == nil {
		return rootCmd
	}
	return cmd
}

// resolveErrorMode applies flag > env > config > default "human"
// precedence for EmitError rendering. Swallows config errors — error
// emission must not itself fail.
func resolveErrorMode(cmd *cobra.Command, f *runtime.Invocation) string {
	if j := cmd.Flag("json"); j != nil && j.Changed {
		return "json"
	}
	if o := cmd.Flag("output"); o != nil && o.Changed {
		return o.Value.String()
	}
	if v, ok := os.LookupEnv("TG_OUTPUT"); ok && v != "" {
		return v
	}
	if f.Config != nil {
		if cfg, err := f.Config(); err == nil && cfg != nil && cfg.Output.Format != nil {
			return *cfg.Output.Format
		}
	}
	return "human"
}
