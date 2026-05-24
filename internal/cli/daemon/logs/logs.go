// Package logs implements "tg daemon logs".
package logs

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/daemon"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	Account   string
	Follow    bool
	Lines     int
	LogFile   string
	IOStreams *ui.IOStreams
}

// New builds the cobra command for "tg daemon logs".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Tail the per-account daemon log",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IOStreams = f.IOStreams
			opts.Account = effectiveAccount(f)
			if runF != nil {
				return runF(opts)
			}
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.Follow, "follow", "f", false, "Follow the log (like tail -F)")
	cmd.Flags().IntVarP(&opts.Lines, "lines", "n", 100, "Number of trailing lines to print before following")
	cmd.Flags().StringVar(&opts.LogFile, "log-file", "", "Override log path (default: meta sidecar or per-account default)")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, SkipAuthCheck: true})
	return cmd
}

// Run prints the trailing N lines and optionally follows.
func Run(ctx context.Context, opts *Options) error {
	if opts.Account == "" {
		return fmt.Errorf("%w: logs requires an account", command.ErrUsage)
	}
	path := opts.LogFile
	if path == "" {
		if meta, err := daemon.LoadMeta(opts.Account); err == nil && meta != nil && meta.LogFile != "" {
			path = meta.LogFile
		}
	}
	if path == "" {
		path = daemon.LogFile(opts.Account)
	}

	if err := printTail(opts.IOStreams.Out, path, opts.Lines); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: log file not found: %s (is the daemon installed?)",
				command.ErrPrecondition, path)
		}
		return err
	}
	if opts.Follow {
		return follow(ctx, opts.IOStreams.Out, path)
	}
	return nil
}

// printTail mirrors `tail -n N`. The whole file is read in once since
// the per-account log is bounded by logrotate (default 10 MB), which
// is well within a single read.
func printTail(w io.Writer, path string, n int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	start := 0
	if len(lines) > n {
		start = len(lines) - n
	}
	for _, line := range lines[start:] {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

// follow tails the file in 300ms polls. Used instead of fsnotify to
// keep the daemon CLI dependency-free; the 300ms cadence is the same
// as cc-connect's followFile.
func follow(ctx context.Context, w io.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			if _, werr := fmt.Fprint(w, line); werr != nil {
				return werr
			}
		}
		if err == io.EOF {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(300 * time.Millisecond):
				reader.Reset(f)
				continue
			}
		}
		if err != nil {
			return err
		}
	}
}

func effectiveAccount(f *runtime.Invocation) string {
	if f.AccountName != "" {
		return f.AccountName
	}
	if f.Account == nil {
		return ""
	}
	acct, err := f.Account("")
	if err != nil || acct == nil {
		return ""
	}
	return acct.Meta.Name
}
