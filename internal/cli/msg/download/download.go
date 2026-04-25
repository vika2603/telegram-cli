// Package download implements "tg msg download <msg-ref>".
package download

import (
	"context"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/cli/complete"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	RawMessageRef string
	Output        string
	Exporter      output.Exporter
	IOStreams     *ui.IOStreams
	Download      actionmessage.DownloadFunc
}

// New builds the cobra command for "tg msg download".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:               "download <msg-ref>",
		Short:             "Download message media",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.MessageRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawMessageRef = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Download = newDownload(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "", "Save path or existing directory (default: media filename)")
	_ = cmd.RegisterFlagCompletionFunc("output", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveDefault
	})
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"message_ref", "path", "name", "mime_type", "bytes"})
	return cmd
}

// Run dispatches the normalized request and renders the saved file.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionmessage.Download(ctx, actionmessage.DownloadRequest{
		RawMessageRef: opts.RawMessageRef,
		Output:        opts.Output,
	}, opts.Download)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderDownload(opts.IOStreams, row)
}

func newDownload(f *runtime.Invocation) actionmessage.DownloadFunc {
	return func(ctx context.Context, q actionmessage.DownloadQuery) (output.DownloadRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.DownloadRow{}, err
		}
		var row output.DownloadRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.DownloadMessageMedia(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
