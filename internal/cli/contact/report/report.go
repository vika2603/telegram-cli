// Package report implements "tg contact report <ref>".
package report

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actioncontact "github.com/vika2603/telegram-cli/internal/action/contact"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	RawRef  string
	Reason  string
	Message string
	Ban     bool
	Yes     bool

	IOStreams *ui.IOStreams
	Prompter  ui.Prompter

	// Report is the closure that performs the actual Telegram call. Production
	// code sets it via newReport; tests stub it directly.
	Report actioncontact.ReportFunc
}

// New builds the cobra command for "tg contact report".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "report <ref>",
		Short: "Report a user, bot, group, or channel to Telegram",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams
			opts.Prompter = f.Prompter
			if runF != nil {
				return runF(opts)
			}
			opts.Report = newReport(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.Reason, "reason", "spam", "Report reason: spam, violence, porn, child-abuse, copyright, fake, drugs, personal-details, geo-irrelevant, other")
	cmd.Flags().StringVar(&opts.Message, "message", "", "Optional comment for report moderation")
	cmd.Flags().BoolVar(&opts.Ban, "ban", false, "Also block the peer after reporting")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	return cmd
}

// Run collects the raw request, delegates validation/mutation, and renders status.
func Run(ctx context.Context, opts *Options) error {
	if err := actioncontact.Report(ctx, actioncontact.ReportRequest{
		RawRef:   opts.RawRef,
		Reason:   opts.Reason,
		Message:  opts.Message,
		Ban:      opts.Ban,
		Yes:      opts.Yes,
		Prompter: opts.Prompter,
	}, opts.Report); err != nil {
		return err
	}
	verb := "reported"
	if opts.Ban {
		verb = "reported+blocked"
	}
	_, _ = fmt.Fprintf(opts.IOStreams.Out, "%s\t%s\n", verb, opts.RawRef)
	return nil
}

// newReport returns the production Report closure that calls the Telegram API.
func newReport(f *runtime.Invocation) actioncontact.ReportFunc {
	return func(ctx context.Context, q actioncontact.ReportQuery) error {
		acct, err := f.Account("")
		if err != nil {
			return err
		}
		if cl, _ := runtime.MaybeDialDaemon(ctx, f, acct); cl != nil {
			defer func() { _ = cl.Close() }()
			_, err := cl.Call(ctx, "contact.report", q)
			return err
		}
		return f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				return telegram.ReportPeer(ctx, api, res, q)
			})
	}
}
