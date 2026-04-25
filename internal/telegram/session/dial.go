package session

import (
	"github.com/gotd/td/telegram"

	"github.com/vika2603/telegram-cli/internal/account"
)

// built bundles the live telegram.Client. Callers close it when
// telegram.Client.Run returns.
type built struct {
	tgCl *telegram.Client
}

func (b *built) close() {}

// buildTelegramClient constructs a telegram.Client. The returned client is not
// yet running; the caller invokes tgCl.Run.
func buildTelegramClient(acct *account.Account, opts Options) (*built, error) {
	return buildTelegramClientWithHandler(acct, opts, nil)
}

// buildTelegramClientWithHandler installs handler into telegram.Options.UpdateHandler
// at client construction time. Passing handler == nil is equivalent to buildTelegramClient.
func buildTelegramClientWithHandler(acct *account.Account, opts Options, handler telegram.UpdateHandler) (*built, error) {
	sess := &account.FileSessionStorage{AccountName: acct.Meta.Name}
	b := &built{}
	tgOpts := telegram.Options{
		Logger:         opts.Logger,
		SessionStorage: sess,
		UpdateHandler:  handler,
		Device:         deviceConfig(opts.Device),
	}
	b.tgCl = telegram.NewClient(opts.APIID, opts.APIHash, tgOpts)
	return b, nil
}

func deviceConfig(d DeviceOptions) telegram.DeviceConfig {
	return telegram.DeviceConfig{
		DeviceModel:    d.Model,
		SystemVersion:  d.SystemVersion,
		AppVersion:     d.AppVersion,
		SystemLangCode: d.SystemLangCode,
		LangCode:       d.LangCode,
	}
}
