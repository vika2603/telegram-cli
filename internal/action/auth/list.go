package auth

import (
	"fmt"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/config"
)

// ListDeps are the local account dependencies used by List.
type ListDeps struct {
	Config       func() (*config.Config, error)
	ListAccounts func() ([]string, error)
	ReadMeta     func(string) (account.Meta, error)
}

// ListResult is the normalized result for `tg auth list`.
type ListResult struct {
	Items     []account.AccountDTO
	WarnNames []string
}

// List loads account DTOs and marks the configured default account.
func List(deps ListDeps) (ListResult, error) {
	if deps.Config == nil {
		return ListResult{}, fmt.Errorf("%w: auth list called without config function", command.ErrPrecondition)
	}
	if deps.ListAccounts == nil {
		return ListResult{}, fmt.Errorf("%w: auth list called without list-accounts function", command.ErrPrecondition)
	}
	if deps.ReadMeta == nil {
		return ListResult{}, fmt.Errorf("%w: auth list called without read-meta function", command.ErrPrecondition)
	}

	cfg, err := deps.Config()
	if err != nil {
		return ListResult{}, err
	}
	names, err := deps.ListAccounts()
	if err != nil {
		return ListResult{}, err
	}
	def := defaultAccount(cfg)

	result := ListResult{
		Items:     make([]account.AccountDTO, 0, len(names)),
		WarnNames: make([]string, 0, len(names)),
	}
	for _, name := range names {
		meta, err := deps.ReadMeta(name)
		if err != nil {
			return ListResult{}, err
		}
		result.Items = append(result.Items, account.DTOFromMeta(meta, name == def))
		result.WarnNames = append(result.WarnNames, name)
	}
	return result, nil
}
