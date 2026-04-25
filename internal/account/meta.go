package account

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
)

type State string

const (
	StateNEW     State = "NEW"
	StateAUTHED  State = "AUTHED"
	StateEXPIRED State = "EXPIRED"
)

// Meta is the on-disk representation of account.json.
type Meta struct {
	Version   int    `json:"version"`
	Name      string `json:"name"`
	State     State  `json:"state"`
	APIID     int    `json:"api_id,omitempty"`
	APIHash   string `json:"api_hash,omitempty"`
	Phone     string `json:"phone,omitempty"`
	CreatedAt int64  `json:"created_at"`
}

// Account is the in-memory handle passed to session.Run / session.Login. It
// carries the loaded Meta plus the on-disk paths so callers do not have to
// recompute them.
type Account struct {
	Meta Meta
	Dir  string // AccountDir(Meta.Name)
	Lock string // AccountDir/account.lock
	Sess string // AccountDir/session.bin
}

// LoadAccount reads account.json for name and returns a populated Account.
func LoadAccount(name string) (*Account, error) {
	m, err := ReadMeta(name)
	if err != nil {
		return nil, err
	}
	return &Account{
		Meta: m,
		Dir:  AccountDir(name),
		Lock: LockFile(name),
		Sess: SessionFile(name),
	}, nil
}

const MetaVersion = 1

var nameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// IsValidName enforces the account-name grammar. "." and ".." are rejected
// explicitly because the regex would otherwise accept them.
func IsValidName(s string) bool {
	if s == "." || s == ".." {
		return false
	}
	return nameRE.MatchString(s)
}

func ReadMeta(name string) (Meta, error) {
	data, err := os.ReadFile(MetaFile(name))
	if errors.Is(err, os.ErrNotExist) {
		return Meta{}, fmt.Errorf("%w: %s", ErrAccountNotFound, name)
	}
	if err != nil {
		return Meta{}, fmt.Errorf("read account meta %s: %w", name, err)
	}
	var m Meta
	if err := json.Unmarshal(data, &m); err != nil {
		return Meta{}, fmt.Errorf("parse account meta %s: %w", name, err)
	}
	if m.Version != MetaVersion {
		return Meta{}, fmt.Errorf("account %s: meta version %d not supported, expected %d", name, m.Version, MetaVersion)
	}
	return m, nil
}

// WriteMeta atomically persists the meta. The write takes account.lock
// first so concurrent writers cannot observe a torn state; a busy lock
// surfaces as ErrBusy.
func WriteMeta(m Meta) error {
	if !IsValidName(m.Name) {
		return fmt.Errorf("invalid account name %q", m.Name)
	}
	if m.Version == 0 {
		m.Version = MetaVersion
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return AtomicWrite(MetaFile(m.Name), LockFile(m.Name), data)
}
