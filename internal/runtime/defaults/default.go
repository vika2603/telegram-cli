// Package defaults wires production defaults onto a runtime.Invocation. The
// separation from command lets leaf-command tests import command without
// pulling in the full dependency chain (bbolt, gotd, etc.) that the
// production closures need.
package defaults

import (
	"go.uber.org/zap"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/logging"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// New returns a fully wired Invocation. IOStreams/Prompter come from
// runtime.NewInvocation (System() defaults). Config/Logger/Account closures
// read f.ConfigPath and f.AccountName, which the root PersistentPreRun
// populates from the --config and --account flags before any RunE fires.
// WithClient delegates to runtime.DefaultWithClient (which calls
// session.Run), keeping the bbolt/gotd lifetime scoped to a single command.
func New(appVersion string) *runtime.Invocation {
	f := runtime.NewInvocation(appVersion)

	f.Config = func() (*config.Config, error) {
		cfg, _, err := config.LoadMerged(config.Config{}, f.ConfigPath)
		if err != nil {
			return nil, err
		}
		return &cfg, nil
	}
	f.Logger = func() (*zap.Logger, func(), error) {
		cfg, err := f.Config()
		if err != nil {
			return nil, nil, err
		}
		return logging.BuildLogger(*cfg)
	}
	f.Account = func(name string) (*account.Account, error) {
		cfg, err := f.Config()
		if err != nil {
			return nil, err
		}
		override := f.AccountName
		if name != "" {
			override = name
		}
		def := ""
		if cfg.DefaultAccount != nil {
			def = *cfg.DefaultAccount
		}
		resolved, err := account.ResolveAccount(override, def)
		if err != nil {
			return nil, err
		}
		return account.LoadAccount(resolved)
	}
	f.WithClient = runtime.DefaultWithClient
	f.Resolver = runtime.DefaultResolver
	f.WithPeers = runtime.DefaultWithPeers

	return f
}
