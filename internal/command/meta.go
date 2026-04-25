package command

import "github.com/spf13/cobra"

// Meta is the typed surface for command annotations the root PersistentPreRun
// consumes. Stored in cobra.Command.Annotations under a "meta:" prefix.
type Meta struct {
	NeedsAccount  bool
	NeedsClient   bool
	SkipAuthCheck bool
	LongRunning   bool

	// AccountFromArg means the slot is taken from the command's first
	// positional argument (optionally — nil arg still falls back to default).
	// When true, the root pre-runE does NOT pre-load f.Account(""); the
	// command body is responsible for calling f.Account(opts.Name) itself
	// and for running its own auth-state gate if applicable.
	AccountFromArg bool
}

const (
	metaKeyNeedsAccount   = "meta:needs-account"
	metaKeyNeedsClient    = "meta:needs-client"
	metaKeySkipAuthCheck  = "meta:skip-auth-check"
	metaKeyLongRunning    = "meta:long-running"
	metaKeyAccountFromArg = "meta:account-from-arg"
)

// SetMeta writes m into cmd.Annotations, preserving any unrelated keys.
func SetMeta(cmd *cobra.Command, m Meta) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	writeBool(cmd, metaKeyNeedsAccount, m.NeedsAccount)
	writeBool(cmd, metaKeyNeedsClient, m.NeedsClient)
	writeBool(cmd, metaKeySkipAuthCheck, m.SkipAuthCheck)
	writeBool(cmd, metaKeyLongRunning, m.LongRunning)
	writeBool(cmd, metaKeyAccountFromArg, m.AccountFromArg)
}

// MetaFrom reads the typed Meta back from cmd.Annotations.
func MetaFrom(cmd *cobra.Command) Meta {
	return Meta{
		NeedsAccount:   readBool(cmd, metaKeyNeedsAccount),
		NeedsClient:    readBool(cmd, metaKeyNeedsClient),
		SkipAuthCheck:  readBool(cmd, metaKeySkipAuthCheck),
		LongRunning:    readBool(cmd, metaKeyLongRunning),
		AccountFromArg: readBool(cmd, metaKeyAccountFromArg),
	}
}

func writeBool(cmd *cobra.Command, key string, v bool) {
	if v {
		cmd.Annotations[key] = "true"
		return
	}
	delete(cmd.Annotations, key)
}

func readBool(cmd *cobra.Command, key string) bool {
	if cmd.Annotations == nil {
		return false
	}
	return cmd.Annotations[key] == "true"
}
