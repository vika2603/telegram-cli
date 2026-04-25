// Package authflow contains helpers for authenticating Telegram accounts.
// These were promoted from internal/cli/account/shared in Task 17.
package authflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/mdp/qrterminal/v3"
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/logging"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// InvocationIO returns f.IOStreams when set, falling back to ui.System().
func InvocationIO(f *runtime.Invocation) *ui.IOStreams {
	if f != nil && f.IOStreams != nil {
		return f.IOStreams
	}
	return ui.System()
}

// CompleteAccountNames drives shell completion for positional args that
// accept an account name. Errors are swallowed — completion must not surface
// them to the shell.
func CompleteAccountNames(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	names, err := account.ListAccounts()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// WriteAccountDTO emits an AccountDTO after a mutating account command
// (add / login). If exp is non-nil the JSON exporter path is used, otherwise
// a human line is written to f.IOStreams.Out. quiet suppresses all output but
// still verifies the meta is readable.
func WriteAccountDTO(f *runtime.Invocation, name, def string, exp output.Exporter, quiet bool) error {
	m, err := account.ReadMeta(name)
	if err != nil {
		return err
	}
	if quiet {
		return nil
	}
	dto := account.DTOFromMeta(m, name == def)
	if exp != nil {
		return exp.Write(f.IOStreams, dto)
	}
	_, _ = fmt.Fprintln(f.IOStreams.Out, dto.Human())
	return nil
}

// CredsIfAvailableFromPtrs returns the api_id/api_hash pair when both pointer
// fields are non-nil and non-zero/empty; otherwise returns (0, "").
// Used by the --no-login path which tolerates a bare slot.
func CredsIfAvailableFromPtrs(apiID *int, apiHash *string) (int, string) {
	id := 0
	hash := ""
	if apiID != nil {
		id = *apiID
	}
	if apiHash != nil {
		hash = *apiHash
	}
	if id != 0 && hash != "" {
		return id, hash
	}
	return 0, ""
}

// ResolveCreds extracts api_id/api_hash from the pointer fields of a merged
// config. Returns ErrPrecondition when either field is absent.
func ResolveCreds(apiID *int, apiHash *string) (int, string, error) {
	id := 0
	hash := ""
	if apiID != nil {
		id = *apiID
	}
	if apiHash != nil {
		hash = *apiHash
	}
	if id == 0 || hash == "" {
		return 0, "", fmt.Errorf("%w: api_id/api_hash not configured (pass --api-id/--api-hash, set TG_API_ID/TG_API_HASH, or add to config.toml)", command.ErrPrecondition)
	}
	return id, hash, nil
}

// RotationCreds resolves rotation credentials by precedence:
// flags > env > config file. Returns (id, hash, shouldRotate, err).
// shouldRotate is true only when BOTH fields differ from the stored values.
// Partial provisioning at the highest-priority tier is a usage error.
func RotationCreds(cmd *cobra.Command, cfgAPIID *int, cfgAPIHash *string, flagAPIID int, flagAPIHash string, curID int, curHash string) (int, string, bool, error) {
	flagIDChanged := cmd.Flags().Changed("api-id")
	flagHashChanged := cmd.Flags().Changed("api-hash")
	return RotationCredsFromInputs(flagIDChanged, flagHashChanged, cfgAPIID, cfgAPIHash, flagAPIID, flagAPIHash, curID, curHash)
}

// RotationCredsFromInputs resolves rotation credentials from already-normalized
// flag state. This keeps action code independent from cobra while preserving
// RotationCreds for existing cobra callers.
func RotationCredsFromInputs(flagIDChanged, flagHashChanged bool, cfgAPIID *int, cfgAPIHash *string, flagAPIID int, flagAPIHash string, curID int, curHash string) (int, string, bool, error) {
	if flagIDChanged != flagHashChanged {
		return 0, "", false, fmt.Errorf("%w: --api-id and --api-hash must be provided together", command.ErrUsage)
	}
	if flagIDChanged && flagHashChanged {
		return ApplyRotation(flagAPIID, flagAPIHash, curID, curHash)
	}

	envIDStr, envHasID := os.LookupEnv("TG_API_ID")
	envHash, envHasHash := os.LookupEnv("TG_API_HASH")
	envIDPresent := envHasID && envIDStr != ""
	envHashPresent := envHasHash && envHash != ""
	if envIDPresent != envHashPresent {
		return 0, "", false, fmt.Errorf("%w: TG_API_ID and TG_API_HASH must be provided together", command.ErrUsage)
	}
	if envIDPresent && envHashPresent {
		id, err := strconv.Atoi(envIDStr)
		if err != nil {
			return 0, "", false, fmt.Errorf("%w: TG_API_ID=%q is not a valid integer", command.ErrUsage, envIDStr)
		}
		return ApplyRotation(id, envHash, curID, curHash)
	}

	// Config-file tier.
	if cfgAPIID != nil && *cfgAPIID != 0 && cfgAPIHash != nil && *cfgAPIHash != "" {
		return ApplyRotation(*cfgAPIID, *cfgAPIHash, curID, curHash)
	}
	return 0, "", false, nil
}

// ApplyRotation validates the new credentials and reports whether rotation
// is needed.
func ApplyRotation(newID int, newHash string, curID int, curHash string) (int, string, bool, error) {
	if newID == 0 || newHash == "" {
		return 0, "", false, fmt.Errorf("%w: api_id/api_hash both required for rotation (got id=%d, hash_empty=%v)", command.ErrUsage, newID, newHash == "")
	}
	return newID, newHash, newID != curID || newHash != curHash, nil
}

// DoLogin runs session.Login or session.LoginQR against the named account.
// It does NOT emit AccountDTO — the enclosing command emits after this
// returns so the DTO reflects the post-login Meta state.
//
// flagAPIID / flagAPIHash carry any --api-id / --api-hash values the caller
// parsed; cmd is used only to normalize whether those flags were changed.
func DoLogin(
	ctx context.Context,
	f *runtime.Invocation,
	cmd *cobra.Command,
	name string,
	qr, force bool,
	flagAPIID int, flagAPIHash string,
) error {
	flagIDChanged := false
	flagHashChanged := false
	if cmd != nil && cmd.Flags() != nil {
		flagIDChanged = cmd.Flags().Changed("api-id")
		flagHashChanged = cmd.Flags().Changed("api-hash")
	}
	return DoLoginWithOptions(ctx, f, LoginOptions{
		Name:           name,
		QR:             qr,
		Force:          force,
		APIID:          flagAPIID,
		APIHash:        flagAPIHash,
		APIIDChanged:   flagIDChanged,
		APIHashChanged: flagHashChanged,
	})
}

// LoginOptions is the cobra-free input for DoLoginWithOptions.
type LoginOptions struct {
	Name           string
	QR             bool
	Force          bool
	APIID          int
	APIHash        string
	APIIDChanged   bool
	APIHashChanged bool
}

// DoLoginWithOptions runs session.Login or session.LoginQR against the named
// account using flag state that has already been normalized by the CLI layer.
func DoLoginWithOptions(ctx context.Context, f *runtime.Invocation, opts LoginOptions) error {
	name := opts.Name
	if !account.IsValidName(name) {
		return fmt.Errorf("%w: invalid account name %q", command.ErrUsage, name)
	}
	acct, err := account.LoadAccount(name)
	if err != nil {
		if errors.Is(err, account.ErrAccountNotFound) {
			return fmt.Errorf("%w: account %q does not exist", command.ErrUsage, name)
		}
		return err
	}
	if acct.Meta.State == account.StateAUTHED && !opts.Force {
		return fmt.Errorf("%w: account %s is already AUTHED (use --force to re-login)", command.ErrPrecondition, name)
	}

	cfg, err := f.Config()
	if err != nil {
		return err
	}

	apiID, apiHash := acct.Meta.APIID, acct.Meta.APIHash
	// Non-force: bare slot (no creds in account.json) falls back to merged config.
	if !opts.Force && (apiID == 0 || apiHash == "") {
		resolvedID, resolvedHash, rerr := ResolveCreds(cfg.APIID, cfg.APIHash)
		if rerr != nil {
			return rerr
		}
		apiID, apiHash = resolvedID, resolvedHash
		acct.Meta.APIID = apiID
		acct.Meta.APIHash = apiHash
	}

	if opts.Force {
		newID, newHash, shouldRotate, rerr := RotationCredsFromInputs(
			opts.APIIDChanged,
			opts.APIHashChanged,
			cfg.APIID,
			cfg.APIHash,
			opts.APIID,
			opts.APIHash,
			apiID,
			apiHash,
		)
		if rerr != nil {
			return rerr
		}
		if shouldRotate {
			acct.Meta.APIID = newID
			acct.Meta.APIHash = newHash
			apiID, apiHash = newID, newHash
		}
		// Interactive prompt when no creds from any source.
		if apiID == 0 || apiHash == "" {
			promptID, promptHash, perr := PromptAPICredentials(InvocationIO(f))
			if perr != nil {
				return perr
			}
			apiID = promptID
			acct.Meta.APIID = apiID
			acct.Meta.APIHash = promptHash
			apiHash = promptHash
		}
	}

	log, logClose, err := logging.BuildLogger(*cfg)
	if err != nil {
		return err
	}
	defer logClose()
	command.WarnLoosePerms(log, acct)

	sessionOpts := runtime.ClientOptsFrom(f, acct)
	sessionOpts.Logger = log
	sessionOpts.APIID = apiID
	sessionOpts.APIHash = apiHash
	authr, display, err := PickAuth(InvocationIO(f), opts.QR)
	if err != nil {
		return err
	}
	if opts.QR {
		return session.LoginQR(ctx, acct, sessionOpts, display)
	}
	return session.Login(ctx, acct, sessionOpts, authr)
}

// PickAuth chooses the (authenticator, display) pair.
func PickAuth(ios *ui.IOStreams, qr bool) (account.UserAuthenticator, account.QRDisplay, error) {
	if ios == nil {
		ios = ui.System()
	}
	if qr {
		d, err := newQRDisplay(ios)
		return nil, d, err
	}
	if os.Getenv("TG_PHONE") != "" || os.Getenv("TG_CODE") != "" ||
		os.Getenv("TG_CODE_CMD") != "" || os.Getenv("TG_2FA_PASSWORD") != "" {
		return newEnvAuth(), nil, nil
	}
	if ios.IsStdinTTY() {
		a, err := newTTYAuth(ios)
		if err != nil {
			return nil, nil, err
		}
		return a, nil, nil
	}
	return nil, nil, fmt.Errorf("%w: no terminal and no env credentials; provide --qr, or set TG_PHONE/TG_CODE (and TG_2FA_PASSWORD if needed), or run in a terminal", command.ErrPrecondition)
}

// PromptAPICredentials reads api_id and api_hash interactively. Fails fast
// with ErrPrecondition when stdin is not a TTY.
func PromptAPICredentials(ios *ui.IOStreams) (int, string, error) {
	if !ios.IsStdinTTY() {
		return 0, "", fmt.Errorf("%w: api_id/api_hash not set and stdin is not a terminal; pass --api-id/--api-hash or set TG_API_ID/TG_API_HASH", command.ErrPrecondition)
	}
	prompter := &ui.SystemPrompter{IO: ios}
	line, err := prompter.Input("Telegram api_id (integer, from https://my.telegram.org)", "")
	if err != nil {
		return 0, "", fmt.Errorf("read api_id: %w", err)
	}
	id, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || id <= 0 {
		return 0, "", fmt.Errorf("%w: api_id must be a positive integer", command.ErrUsage)
	}
	hash, err := prompter.Password("Telegram api_hash")
	if err != nil {
		return 0, "", fmt.Errorf("read api_hash: %w", err)
	}
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return 0, "", fmt.Errorf("%w: api_hash is required", command.ErrUsage)
	}
	return id, hash, nil
}

// ttyAuth prompts on stderr and reads from the tty.
type ttyAuth struct {
	io       *ui.IOStreams
	prompter ui.Prompter
}

func newTTYAuth(ios *ui.IOStreams) (*ttyAuth, error) {
	if !ios.IsStdinTTY() {
		return nil, fmt.Errorf("%w: stdin is not a terminal", command.ErrPrecondition)
	}
	return &ttyAuth{io: ios, prompter: &ui.SystemPrompter{IO: ios}}, nil
}

func (t *ttyAuth) Phone(_ context.Context) (string, error) {
	line, err := t.prompter.Input("Phone (with country code, e.g. +8613800138000)", "")
	return strings.TrimSpace(line), err
}

func (t *ttyAuth) Code(_ context.Context, s account.SentCode) (string, error) {
	line, err := t.prompter.Input(fmt.Sprintf("Code (sent via %s, timeout %s)", s.Type, s.Timeout), "")
	return strings.TrimSpace(line), err
}

func (t *ttyAuth) Password(_ context.Context) (string, error) {
	return t.prompter.Password("2FA password")
}

func (t *ttyAuth) AcceptTOS(_ context.Context, tos account.TermsOfService) error {
	_, _ = fmt.Fprintln(t.io.ErrOut, "Terms of Service:")
	_, _ = fmt.Fprintln(t.io.ErrOut, tos.Text)
	ok, err := t.prompter.Confirm("Accept Terms of Service?", false)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("terms of service not accepted")
	}
	return nil
}

// envAuth reads credentials from environment variables.
type envAuth struct{}

func newEnvAuth() *envAuth { return &envAuth{} }

func (e *envAuth) Phone(_ context.Context) (string, error) {
	v := os.Getenv("TG_PHONE")
	if v == "" {
		return "", fmt.Errorf("%w: TG_PHONE not set", command.ErrPrecondition)
	}
	return v, nil
}

func (e *envAuth) Code(ctx context.Context, _ account.SentCode) (string, error) {
	if v := os.Getenv("TG_CODE"); v != "" {
		return v, nil
	}
	if cmd := os.Getenv("TG_CODE_CMD"); cmd != "" {
		out, err := exec.CommandContext(ctx, "sh", "-c", cmd).Output()
		if err != nil {
			return "", fmt.Errorf("TG_CODE_CMD: %w", err)
		}
		return strings.TrimSpace(string(out)), nil
	}
	return "", fmt.Errorf("%w: neither TG_CODE nor TG_CODE_CMD set", command.ErrPrecondition)
}

func (e *envAuth) Password(_ context.Context) (string, error) {
	v := os.Getenv("TG_2FA_PASSWORD")
	if v == "" {
		return "", fmt.Errorf("%w: TG_2FA_PASSWORD not set (account has 2FA)", command.ErrPrecondition)
	}
	return v, nil
}

func (e *envAuth) AcceptTOS(_ context.Context, _ account.TermsOfService) error {
	return fmt.Errorf("%w: cannot accept TOS non-interactively", command.ErrPrecondition)
}

// qrDisplay renders an ASCII QR code to stderr.
type qrDisplay struct {
	io *ui.IOStreams
}

func newQRDisplay(ios *ui.IOStreams) (*qrDisplay, error) {
	if !ios.IsStderrTTY() {
		return nil, fmt.Errorf("%w: stderr is not a terminal, cannot render QR", command.ErrPrecondition)
	}
	return &qrDisplay{io: ios}, nil
}

func (q *qrDisplay) Show(_ context.Context, url string) error {
	_, _ = fmt.Fprintln(q.io.ErrOut, "Scan this QR from Telegram -> Settings -> Devices -> Link Desktop Device:")
	qrterminal.GenerateHalfBlock(url, qrterminal.L, q.io.ErrOut)
	return nil
}

func (q *qrDisplay) Refresh(ctx context.Context, url string) error { return q.Show(ctx, url) }

func (q *qrDisplay) Done(_ context.Context, accepted bool) {
	if accepted {
		_, _ = fmt.Fprintln(q.io.ErrOut, "Login successful.")
	} else {
		_, _ = fmt.Fprintln(q.io.ErrOut, "Login cancelled or failed.")
	}
}
