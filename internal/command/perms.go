package command

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"go.uber.org/zap"

	"github.com/vika2603/telegram-cli/internal/account"
)

// WarnLoosePerms emits a warn if either account.json or session.bin has
// permission bits looser than 0600. Non-existent session.bin is not a warning
// (fresh NEW account). Stat errors other than ENOENT log at warn and do not
// abort the caller. This is advisory — we do not refuse to load.
func WarnLoosePerms(log *zap.Logger, acct *account.Account) {
	checkFile(log, acct.Meta.Name, "meta", account.MetaFile(acct.Meta.Name))
	checkFile(log, acct.Meta.Name, "session", acct.Sess)
}

// WarnLoosePermsByName is the logger-free fallback for commands that do not
// build a full RunCtx. Emits plain "WARN: ..." lines to stderr.
func WarnLoosePermsByName(stderr io.Writer, name string) {
	for _, kind := range []struct {
		kind string
		path string
	}{
		{"meta", account.MetaFile(name)},
		{"session", account.SessionFile(name)},
	} {
		info, err := os.Stat(kind.path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			_, _ = fmt.Fprintf(stderr, "WARN: perm stat failed for %s %s: %v\n",
				kind.kind, kind.path, err)
			continue
		}
		mode := info.Mode().Perm()
		if mode&0o077 != 0 {
			_, _ = fmt.Fprintf(stderr, "WARN: file permissions looser than 0600: account=%s kind=%s path=%s mode=%s\n",
				name, kind.kind, kind.path, mode)
		}
	}
}

func checkFile(log *zap.Logger, acctName, kind, path string) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return
		}
		log.Warn("perm stat failed",
			zap.String("account", acctName),
			zap.String("kind", kind),
			zap.String("path", path),
			zap.Error(err))
		return
	}
	mode := info.Mode().Perm()
	if mode&0o077 != 0 {
		log.Warn("file permissions looser than 0600",
			zap.String("account", acctName),
			zap.String("kind", kind),
			zap.String("path", path),
			zap.String("mode", mode.String()))
	}
}
