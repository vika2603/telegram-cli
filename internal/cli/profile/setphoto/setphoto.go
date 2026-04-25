// Package setphoto implements "tg profile set-photo <path>".
package setphoto

import (
	"context"
	"fmt"
	"io"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionprofile "github.com/vika2603/telegram-cli/internal/action/profile"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	Path string

	IOStreams *ui.IOStreams
	Exporter  output.Exporter
	Stdin     io.Reader

	// Upload is the closure that performs the actual Telegram call. Production
	// code sets it via newUpload; tests stub it directly.
	Upload actionprofile.SetPhotoFunc
}

// New builds the cobra command for "tg profile set-photo".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "set-photo <path>",
		Short: "Set profile photo ('-' reads stdin bytes)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Path = args[0]
			opts.IOStreams = f.IOStreams
			opts.Stdin = f.IOStreams.In
			if runF != nil {
				return runF(opts)
			}
			opts.Upload = newUpload(f)
			return Run(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "photo_id"})
	return cmd
}

// Run validates options and dispatches the Upload call.
func Run(ctx context.Context, opts *Options) error {
	if opts.Upload == nil {
		return fmt.Errorf("%w: internal error: profile photo upload function is not configured", command.ErrPrecondition)
	}
	stdin := opts.Stdin
	if stdin == nil && opts.IOStreams != nil {
		stdin = opts.IOStreams.In
	}
	row, err := actionprofile.SetPhoto(ctx, actionprofile.SetPhotoRequest{Path: opts.Path, Stdin: stdin}, opts.Upload)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.WriteProfileJSON(opts.IOStreams.Out, row)
}

// newUpload returns the production Upload closure that calls the Telegram API.
func newUpload(f *runtime.Invocation) actionprofile.SetPhotoFunc {
	return func(ctx context.Context, path string, stdin io.Reader) (output.ProfileRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.ProfileRow{}, err
		}
		var row output.ProfileRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
				var err error
				row, err = telegram.UploadProfilePhoto(ctx, api, path, stdin)
				return err
			})
		return row, err
	}
}
