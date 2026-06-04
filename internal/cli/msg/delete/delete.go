// Package deletecmd implements "tg msg delete <msg-ref> [<msg-ref>...]".
package deletecmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/cli/complete"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	RawMessageRefs []string
	Revoke         bool
	Yes            bool

	Prompter  ui.Prompter
	IOStreams *ui.IOStreams
	Delete    actionmessage.DeleteFunc
}

// New builds the cobra command for "tg msg delete".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:               "delete <msg-ref> [<msg-ref>...]",
		Short:             "Delete messages",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: complete.MessageRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawMessageRefs = args
			opts.IOStreams = f.IOStreams
			opts.Prompter = f.Prompter
			if runF != nil {
				return runF(opts)
			}
			opts.Delete = newDelete(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.Revoke, "revoke", false, "Delete for everyone (not just self-side)")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	return cmd
}

// Run dispatches the normalized request and renders the result.
func Run(ctx context.Context, opts *Options) error {
	result, err := actionmessage.Delete(ctx, actionmessage.DeleteRequest{
		RawMessageRefs: opts.RawMessageRefs,
		Revoke:         opts.Revoke,
		Yes:            opts.Yes,
		Prompter:       opts.Prompter,
	}, opts.Delete)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(opts.IOStreams.ErrOut, "%s %d message(s)\n", result.Verb, result.Count)
	_, _ = fmt.Fprintf(opts.IOStreams.Out, "%s\t%d\n", result.Verb, result.Count)
	return nil
}

// newDelete returns the production Delete closure that calls the Telegram API.
func newDelete(f *runtime.Invocation) actionmessage.DeleteFunc {
	return func(ctx context.Context, q actionmessage.DeleteQuery) (int, error) {
		acct, err := f.Account("")
		if err != nil {
			return 0, err
		}
		if cl, _ := runtime.MaybeDialDaemon(ctx, f, acct); cl != nil {
			defer func() { _ = cl.Close() }()
			raw, err := cl.Call(ctx, "msg.delete", q)
			if err != nil {
				return 0, err
			}
			return decodeDeleteCount(raw, len(q.IDs)), nil
		}
		var affected int
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				affected, err = telegram.DeleteMessages(ctx, api, res, q)
				return err
			})
		return affected, err
	}
}

// decodeDeleteCount reads the affected-message count from a daemon msg.delete
// response. A current daemon returns the integer count; an older one returns a
// bare "true" ack that won't decode, in which case we fall back to the
// requested count so delete still works across a version skew.
func decodeDeleteCount(raw json.RawMessage, requested int) int {
	var affected int
	if err := json.Unmarshal(raw, &affected); err != nil {
		return requested
	}
	return affected
}
