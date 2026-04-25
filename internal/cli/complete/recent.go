// Package complete contains shared cobra completion helpers.
package complete

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func PeerRefs(f *runtime.Invocation) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return recentPeerSuggestions(cmd, f, toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}

func MessageRefs(f *runtime.Invocation) cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return recentMessageSuggestions(cmd, f, toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}

func recentPeerSuggestions(cmd *cobra.Command, f *runtime.Invocation, query string) []string {
	store, closeFn := openRecentStore(cmd, f)
	if store == nil {
		return nil
	}
	defer closeFn()
	rows, err := store.ListRecentPeers(query, 30)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Ref == "" {
			continue
		}
		out = append(out, row.Ref+"\t"+peerDescription(row))
	}
	return out
}

func recentMessageSuggestions(cmd *cobra.Command, f *runtime.Invocation, query string) []string {
	store, closeFn := openRecentStore(cmd, f)
	if store == nil {
		return nil
	}
	defer closeFn()
	rows, err := store.ListRecentMessages(query, 30)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Ref == "" {
			continue
		}
		out = append(out, row.Ref+"\t"+messageDescription(row))
	}
	return out
}

func openRecentStore(cmd *cobra.Command, f *runtime.Invocation) (*account.PeerStore, func()) {
	name := accountNameForCompletion(cmd, f)
	if name == "" {
		return nil, func() {}
	}
	store, err := account.OpenRecentStoreReadOnly(name)
	if err != nil {
		return nil, func() {}
	}
	return store, func() {}
}

func accountNameForCompletion(cmd *cobra.Command, f *runtime.Invocation) string {
	if f != nil && f.AccountName != "" {
		return f.AccountName
	}
	if cmd != nil && cmd.Root() != nil {
		if flag := cmd.Root().PersistentFlags().Lookup("account"); flag != nil && flag.Value.String() != "" {
			return flag.Value.String()
		}
	}
	cfgPath := ""
	if f != nil && f.ConfigPath != "" {
		cfgPath = f.ConfigPath
	}
	if cmd != nil && cmd.Root() != nil {
		if flag := cmd.Root().PersistentFlags().Lookup("config"); flag != nil && flag.Value.String() != "" {
			cfgPath = flag.Value.String()
		}
	}
	cfg, _, err := config.LoadMerged(config.Config{}, cfgPath)
	if err != nil || cfg.DefaultAccount == nil {
		return ""
	}
	return *cfg.DefaultAccount
}

func peerDescription(row account.RecentPeer) string {
	parts := make([]string, 0, 3)
	if row.Title != "" {
		parts = append(parts, row.Title)
	}
	if row.Username != "" && row.Title != "@"+row.Username {
		parts = append(parts, "@"+row.Username)
	}
	if row.Kind != "" {
		parts = append(parts, row.Kind)
	}
	if row.ID != 0 {
		parts = append(parts, fmt.Sprintf("id:%d", row.ID))
	}
	return strings.Join(parts, " ")
}

func messageDescription(row account.RecentMessage) string {
	if row.Text != "" {
		return row.Text
	}
	if row.PeerRef != "" {
		return row.PeerRef
	}
	if row.MessageID != 0 {
		return strconv.Itoa(row.MessageID)
	}
	return ""
}
