// Package root builds the top-level tg cobra command: groups, global
// flags, the Meta-driven PersistentPreRunE, and the subcommand tree.
package root

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/account"
	authcmd "github.com/vika2603/telegram-cli/internal/cli/auth"
	authlogincmd "github.com/vika2603/telegram-cli/internal/cli/auth/login"
	authlogoutcmd "github.com/vika2603/telegram-cli/internal/cli/auth/logout"
	chatcmd "github.com/vika2603/telegram-cli/internal/cli/chat"
	chatshowcmd "github.com/vika2603/telegram-cli/internal/cli/chat/show"
	"github.com/vika2603/telegram-cli/internal/cli/completion"
	configcmd "github.com/vika2603/telegram-cli/internal/cli/config"
	contactcmd "github.com/vika2603/telegram-cli/internal/cli/contact"
	daemoncmd "github.com/vika2603/telegram-cli/internal/cli/daemon"
	digestcmd "github.com/vika2603/telegram-cli/internal/cli/digest"
	inboxcmd "github.com/vika2603/telegram-cli/internal/cli/inbox"
	mecmd "github.com/vika2603/telegram-cli/internal/cli/me"
	msgcmd "github.com/vika2603/telegram-cli/internal/cli/msg"
	msglistcmd "github.com/vika2603/telegram-cli/internal/cli/msg/list"
	msgsendcmd "github.com/vika2603/telegram-cli/internal/cli/msg/send"
	passwordcmd "github.com/vika2603/telegram-cli/internal/cli/password"
	profilecmd "github.com/vika2603/telegram-cli/internal/cli/profile"
	replycmd "github.com/vika2603/telegram-cli/internal/cli/reply"
	searchcmd "github.com/vika2603/telegram-cli/internal/cli/search"
	sessioncmd "github.com/vika2603/telegram-cli/internal/cli/session"
	watchcmd "github.com/vika2603/telegram-cli/internal/cli/watch"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
)

// New constructs the root command tree. Invocation is injected for
// testability.
func New(f *runtime.Invocation) *cobra.Command {
	cobra.EnableTraverseRunHooks = true

	cmd := &cobra.Command{
		Use:               "tg",
		Short:             "A script-friendly, JSON-first Telegram CLI",
		Version:           f.AppVersion,
		SilenceErrors:     true,
		SilenceUsage:      true,
		PersistentPreRunE: newRootPreRun(f),
	}

	// Global flags. Per-command flags (--json, --jq, --template, --api-id,
	// --api-hash, ...) live on their own commands and are registered by
	// output.AddJSONFlags or the verb package itself.
	pf := cmd.PersistentFlags()
	pf.String("account", "", "Account name (overrides default)")
	pf.String("config", "", "Config file path (default: ~/.config/tg/config.toml)")
	pf.String("output", "", "Output format: human | json")
	pf.String("color", "", "Color: auto | always | never")
	pf.Bool("wait", false, "Wait on FLOOD_WAIT instead of failing")
	pf.Bool("no-wait", false, "Fail on FLOOD_WAIT")
	pf.Int("flood-wait-max", 30, "Max seconds to wait on FLOOD_WAIT")
	pf.Bool("quiet", false, "Suppress stdout")
	pf.Bool("no-daemon", false, "Bypass the per-account daemon and dial MTProto directly")

	fixed := func(values ...string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return values, cobra.ShellCompDirectiveNoFileComp
		}
	}
	_ = cmd.RegisterFlagCompletionFunc("output", fixed("human", "json"))
	_ = cmd.RegisterFlagCompletionFunc("color", fixed("auto", "always", "never"))

	cmd.AddGroup(
		&cobra.Group{ID: "frequent", Title: "Frequent commands:"},
		&cobra.Group{ID: "core", Title: "Core commands:"},
		&cobra.Group{ID: "setup", Title: "Setup commands:"},
	)
	cmd.SetHelpCommandGroupID("setup")
	cmd.SetCompletionCommandGroupID("setup")

	cmd.AddCommand(authcmd.New(f))
	cmd.AddCommand(sessioncmd.New(f))
	cmd.AddCommand(passwordcmd.New(f))
	cmd.AddCommand(configcmd.New(f))
	cmd.AddCommand(daemoncmd.New(f))
	cmd.AddCommand(completion.New(f))
	cmd.AddCommand(frequentCommands(f)...)
	cmd.AddCommand(chatcmd.New(f))
	cmd.AddCommand(msgcmd.New(f))
	cmd.AddCommand(searchcmd.New(f))
	cmd.AddCommand(mecmd.New(f, nil))
	cmd.AddCommand(contactcmd.New(f))
	cmd.AddCommand(profilecmd.New(f))

	return cmd
}

func frequentCommands(f *runtime.Invocation) []*cobra.Command {
	login := authlogincmd.New(f, nil)
	login.GroupID = "frequent"

	logout := authlogoutcmd.New(f, nil)
	logout.GroupID = "frequent"

	send := msgsendcmd.New(f, nil)
	send.GroupID = "frequent"

	reply := replycmd.New(f, nil)
	reply.GroupID = "frequent"

	inbox := inboxcmd.New(f, nil)
	inbox.GroupID = "frequent"

	read := msglistcmd.New(f, nil)
	read.Use = "read <ref>"
	read.Short = "Read message history"
	read.GroupID = "frequent"

	digest := digestcmd.New(f, nil)
	digest.GroupID = "frequent"

	resolve := chatshowcmd.New(f, nil)
	resolve.Use = "resolve <ref>"
	resolve.Short = "Resolve a user, chat, or channel"
	resolve.GroupID = "frequent"

	watch := watchcmd.New(f, nil)
	watch.GroupID = "frequent"

	return []*cobra.Command{login, logout, send, reply, inbox, read, digest, resolve, watch}
}

// newRootPreRun builds the PersistentPreRunE closure that consumes Meta
// and mirrors --config / --account into Invocation fields so Invocation closures
// stay cobra-free.
func newRootPreRun(f *runtime.Invocation) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		if v, err := cmd.Flags().GetString("color"); err == nil {
			switch v {
			case "always":
				f.IOStreams.SetColorEnabled(true)
			case "never":
				f.IOStreams.SetColorEnabled(false)
			}
		}
		f.ConfigPath, _ = cmd.Flags().GetString("config")
		f.AccountName, _ = cmd.Flags().GetString("account")
		f.NoDaemon, _ = cmd.Flags().GetBool("no-daemon")

		m := command.MetaFrom(cmd)
		if m.AccountFromArg {
			// Command body owns slot resolution; skip preload + auth gate.
			return nil
		}
		if !m.NeedsAccount && !m.NeedsClient {
			return nil
		}
		if f.Account == nil {
			return command.ErrPrecondition
		}
		acct, err := f.Account("")
		if err != nil {
			return err
		}
		cmd.SetContext(withAccount(cmd.Context(), acct))

		if !m.SkipAuthCheck && acct != nil && acct.Meta.State != account.StateAUTHED {
			return session.ErrAuth
		}
		return nil
	}
}

// Use a package-private type to avoid context-key collision.
type accountCtxKeyType struct{}

var accountCtxKey = accountCtxKeyType{}

func withAccount(ctx context.Context, acct *account.Account) context.Context {
	return context.WithValue(ctx, accountCtxKey, acct)
}

// AccountFromCtx pulls the account stashed by the root PersistentPreRunE.
// Returns nil if no account is attached.
func AccountFromCtx(ctx context.Context) *account.Account {
	if v := ctx.Value(accountCtxKey); v != nil {
		return v.(*account.Account)
	}
	return nil
}
