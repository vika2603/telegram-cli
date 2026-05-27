// Package setusername implements "tg profile set-username <username>".
package setusername

import (
	"context"
	"encoding/json"
	"fmt"

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
	Username string

	IOStreams *ui.IOStreams
	Exporter  output.Exporter

	// Update is the closure that performs the actual Telegram call. Production
	// code sets it via newUpdate; tests stub it directly.
	Update actionprofile.SetUsernameFunc
}

// New builds the cobra command for "tg profile set-username".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "set-username <username>",
		Short: "Set or clear my public username",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Username = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Update = newUpdate(f)
			return Run(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "username"})
	return cmd
}

// Run validates options and dispatches the Update call.
func Run(ctx context.Context, opts *Options) error {
	if opts.Update == nil {
		return fmt.Errorf("%w: internal error: profile username update function is not configured", command.ErrPrecondition)
	}
	row, err := actionprofile.SetUsername(ctx, opts.Username, opts.Update)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.WriteProfileJSON(opts.IOStreams.Out, row)
}

// newUpdate returns the production Update closure that calls the Telegram API.
func newUpdate(f *runtime.Invocation) actionprofile.SetUsernameFunc {
	return func(ctx context.Context, u string) (output.ProfileRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.ProfileRow{}, err
		}
		if cl, _ := runtime.MaybeDialDaemon(ctx, f, acct); cl != nil {
			defer func() { _ = cl.Close() }()
			raw, err := cl.Call(ctx, "profile.set_username", map[string]string{"username": u})
			if err != nil {
				return output.ProfileRow{}, err
			}
			var row output.ProfileRow
			if err := json.Unmarshal(raw, &row); err != nil {
				return output.ProfileRow{}, err
			}
			return row, nil
		}

		var row output.ProfileRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
				var err error
				row, err = telegram.UpdateProfileUsername(ctx, api, u)
				return err
			})
		return row, err
	}
}
