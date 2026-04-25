// Package logout implements "tg auth logout [name] [--purge]".
package logout

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/account"
	actionauth "github.com/vika2603/telegram-cli/internal/action/auth"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	Name  string
	Purge bool
	Yes   bool

	F        *runtime.Invocation
	Exporter output.Exporter

	// Do runs inside WithClient: AuthLogOut + DeleteSession + WriteMeta(state=NEW).
	// The outer Run loop calls account.RemoveAccount after WithClient returns,
	// only when opts.Purge is set. That keeps directory removal outside the
	// live gotd / bbolt file handles.
	Do func(ctx context.Context, a DoArgs) error
}

// DoArgs are passed into the Do closure by Run.
type DoArgs = actionauth.LogoutDoArgs

// New builds the cobra command for "tg auth logout [name]".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{F: f}
	cmd := &cobra.Command{
		Use:   "logout [name]",
		Short: "Revoke the Telegram session and optionally purge the local slot",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				opts.Name = args[0]
			}
			if runF != nil {
				return runF(opts)
			}
			opts.Do = actionauth.DefaultLogoutDo
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.Purge, "purge", false,
		"Also remove the local account directory (peers.db, session.bin, account.json)")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip destructive confirmation prompt")
	// AccountFromArg: positional [name] selects the slot. Root pre-runE must
	// NOT pre-load the default slot — the command body resolves it via
	// f.Account(opts.Name).
	command.SetMeta(cmd, command.Meta{AccountFromArg: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "name", "purged", "default_cleared"})
	return cmd
}

// Run executes the logout operation.
func Run(ctx context.Context, opts *Options) error {
	result, err := actionauth.Logout(ctx, actionauth.LogoutRequest{
		Name:  opts.Name,
		Purge: opts.Purge,
		Yes:   opts.Yes,
	}, actionauth.LogoutDeps{
		ResolveAccount: opts.F.Account,
		Config:         opts.F.Config,
		Confirm: func(message string, yes bool) error {
			return ui.ConfirmDestructive(opts.F.Prompter, message, yes)
		},
		RunAuthed: func(ctx context.Context, acct *account.Account, do actionauth.LogoutDoFunc) error {
			clientOpts := runtime.ClientOptsFrom(opts.F, acct)
			return opts.F.WithClient(ctx, acct, clientOpts,
				func(ctx context.Context, cl session.Client) error {
					return do(ctx, DoArgs{AcctName: acct.Meta.Name, API: tg.NewClient(cl.Invoker())})
				},
			)
		},
		DeleteSession: account.DeleteSession,
		WriteMeta:     account.WriteMeta,
		RemoveAccount: account.RemoveAccount,
		ClearDefault: func() (actionauth.ClearDefaultResult, error) {
			path := opts.F.ConfigPath
			if path == "" {
				path = account.ConfigFile()
			}
			err := config.UnsetDefaultAccount(path)
			return actionauth.ClearDefaultResult{Path: path, Cleared: err == nil}, err
		},
		Do: opts.Do,
	})
	if err != nil {
		return err
	}
	for _, warning := range result.Warnings {
		_, _ = fmt.Fprintln(opts.F.IOStreams.ErrOut, warning)
	}
	row := result.Row
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.F.IOStreams, row)
	}
	// Non-JSON fallback: human one-liner.
	_, _ = fmt.Fprintf(opts.F.IOStreams.Out, "logged out %s", row.Name)
	if opts.Purge {
		_, _ = fmt.Fprint(opts.F.IOStreams.Out, " (purged)")
	}
	if row.DefaultCleared {
		_, _ = fmt.Fprint(opts.F.IOStreams.Out, " [default cleared]")
	}
	_, _ = fmt.Fprintln(opts.F.IOStreams.Out)
	return nil
}
