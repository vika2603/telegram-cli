package session

import (
	"github.com/gotd/td/telegram"
	"go.etcd.io/bbolt"

	"github.com/vika2603/telegram-cli/internal/account"
)

// built bundles the live telegram.Client together with the bbolt handles it
// keeps open and the stores that wrap them. Callers close it when
// telegram.Client.Run returns.
type built struct {
	tgCl    *telegram.Client
	peersDB *bbolt.DB
	pStore  *account.PeerStore
}

func (b *built) close() {
	if b.peersDB != nil {
		_ = b.peersDB.Close()
		b.peersDB = nil
	}
}

// dbFlags selects which per-account bbolt DBs buildTelegramClient opens.
type dbFlags uint8

const (
	dbPeers dbFlags = 1 << iota
)

// buildTelegramClient constructs a telegram.Client and opens the bbolt DBs
// named by flags. The returned client is not yet running; the caller invokes
// tgCl.Run and must close the returned value when Run returns.
func buildTelegramClient(acct *account.Account, opts Options, flags dbFlags) (*built, error) {
	return buildTelegramClientWithHandler(acct, opts, flags, nil)
}

// buildTelegramClientWithHandler installs handler into telegram.Options.UpdateHandler
// at client construction time. Passing handler == nil is equivalent to buildTelegramClient.
func buildTelegramClientWithHandler(acct *account.Account, opts Options, flags dbFlags, handler telegram.UpdateHandler) (*built, error) {
	sess := &account.FileSessionStorage{AccountName: acct.Meta.Name}
	b := &built{}
	if flags&dbPeers != 0 {
		peersDB, err := account.OpenPeersDB(acct.Meta.Name)
		if err != nil {
			return nil, err
		}
		b.peersDB = peersDB
		b.pStore = account.NewPeerStore(peersDB)
	}
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
