// Package login implements the "tg auth login" command.
package login

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/account"
	actionauth "github.com/vika2603/telegram-cli/internal/action/auth"
	"github.com/vika2603/telegram-cli/internal/authflow"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// DoLoginFunc is the signature of the login driver so tests can stub it.
type DoLoginFunc = actionauth.LoginFunc

// Options holds flag values and injectable dependencies for "auth login".
type Options struct {
	Name           string
	Force          bool
	QR             bool
	NoLogin        bool
	APIID          int
	APIHash        string
	APIIDChanged   bool
	APIHashChanged bool

	F        *runtime.Invocation
	Exporter output.Exporter

	// DoLogin is the login driver; defaults to authflow.DoLogin in production.
	// Tests inject a stub to avoid live MTProto.
	DoLogin DoLoginFunc
}

// New constructs the cobra.Command for "auth login".
// When runF is nil, production logic (Run) is used.
// Tests pass a capture lambda to verify flag parsing without touching disk.
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{F: f}

	cmd := &cobra.Command{
		Use:               "login <name>",
		Short:             "Create-if-missing account slot and log in",
		Args:              cobra.ExactArgs(1),
		SilenceUsage:      true,
		ValidArgsFunction: authflow.CompleteAccountNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
			if v, err := cmd.Flags().GetInt("api-id"); err == nil {
				opts.APIID = v
			}
			if v, err := cmd.Flags().GetString("api-hash"); err == nil {
				opts.APIHash = v
			}
			opts.APIIDChanged = cmd.Flags().Changed("api-id")
			opts.APIHashChanged = cmd.Flags().Changed("api-hash")
			if opts.DoLogin == nil {
				opts.DoLogin = func(ctx context.Context, o authflow.LoginOptions) error {
					return authflow.DoLoginWithOptions(ctx, f, o)
				}
			}
			if runF != nil {
				return runF(opts)
			}
			return Run(cmd.Context(), cmd, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.Force, "force", false, "Re-login even if already AUTHED")
	cmd.Flags().BoolVar(&opts.QR, "qr", false, "Use QR login instead of code login")
	cmd.Flags().BoolVar(&opts.NoLogin, "no-login", false, "Create the slot only; do not run login")
	cmd.Flags().Int("api-id", 0, "Telegram API ID")
	cmd.Flags().String("api-hash", "", "Telegram API hash")

	command.SetMeta(cmd, command.Meta{SkipAuthCheck: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"name", "state", "api_id", "default"})

	return cmd
}

// Run executes the auth login action and renders the resulting account DTO.
func Run(ctx context.Context, cmd *cobra.Command, opts *Options) error {
	if cmd != nil && cmd.Flags() != nil {
		if flag := cmd.Flags().Lookup("api-id"); flag != nil {
			opts.APIIDChanged = cmd.Flags().Changed("api-id")
		}
		if flag := cmd.Flags().Lookup("api-hash"); flag != nil {
			opts.APIHashChanged = cmd.Flags().Changed("api-hash")
		}
	}

	dto, err := actionauth.Login(ctx, actionauth.LoginRequest{
		Name:           opts.Name,
		Force:          opts.Force,
		QR:             opts.QR,
		NoLogin:        opts.NoLogin,
		APIID:          opts.APIID,
		APIHash:        opts.APIHash,
		APIIDChanged:   opts.APIIDChanged,
		APIHashChanged: opts.APIHashChanged,
	}, actionauth.LoginDeps{
		Config:     opts.F.Config,
		ReadMeta:   account.ReadMeta,
		AddAccount: account.AddAccount,
		PromptAPICredentials: func() (int, string, error) {
			return authflow.PromptAPICredentials(authflow.InvocationIO(opts.F))
		},
		DoLogin: opts.DoLogin,
	})
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.F.IOStreams, dto)
	}
	_, _ = opts.F.IOStreams.Out.Write([]byte(dto.Human() + "\n"))
	return nil
}
