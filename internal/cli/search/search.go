// Package search hosts the "tg search" command group. `tg search <q>`
// (no subcommand) shims to `tg search msg <q>`.
package search

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/cli/search/chat"
	"github.com/vika2603/telegram-cli/internal/cli/search/msg"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// New builds the cobra command group for "tg search".
func New(f *runtime.Invocation) *cobra.Command {
	bareFlags := &bareMsgFlags{Limit: 20, Order: "desc"}
	cmd := &cobra.Command{
		Use:     "search <query>",
		Short:   "Unified discovery (defaults to messages)",
		GroupID: "core",
		Args:    cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			// Build a fresh command for the shim path so this closure
			// does not mutate the registered subcommand's arg state.
			shim := msg.New(f, nil)
			shim.SetArgs(bareFlags.args(cmd, args))
			return shim.Execute()
		},
	}
	bareFlags.add(cmd)
	cmd.AddCommand(msg.New(f, nil))
	cmd.AddCommand(chat.New(f, nil))
	return cmd
}

type bareMsgFlags struct {
	In             string
	Filter         string
	From           string
	MinDate        string
	MaxDate        string
	BroadcastsOnly bool
	GroupsOnly     bool
	UsersOnly      bool
	Missed         bool
	Limit          int
	Order          string
	JSON           string
	JQ             string
	Template       string
}

func (f *bareMsgFlags) add(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&f.In, "in", "", "Narrow to one chat (ref)")
	flags.StringVar(&f.Filter, "filter", "",
		"Content filter: photos|video|document|voice|music|gif|url|pinned|geo|my-mentions|round-video|round-voice|phone-calls|chat-photos|contacts|poll|photo-video")
	flags.StringVar(&f.From, "from", "", "Filter by sender (in-chat only)")
	flags.StringVar(&f.MinDate, "min-date", "", "RFC3339 lower bound")
	flags.StringVar(&f.MaxDate, "max-date", "", "RFC3339 upper bound")
	flags.BoolVar(&f.BroadcastsOnly, "broadcasts-only", false, "Global-search only: channels")
	flags.BoolVar(&f.GroupsOnly, "groups-only", false, "Global-search only: groups")
	flags.BoolVar(&f.UsersOnly, "users-only", false, "Global-search only: 1:1 chats")
	flags.BoolVar(&f.Missed, "missed", false, "With --filter phone-calls: missed only")
	flags.IntVar(&f.Limit, "limit", 20, "Max results (cap 1000)")
	flags.StringVar(&f.Order, "order", "desc", "asc|desc")
	flags.StringVar(&f.JSON, "json", "", "emit JSON; value is comma-separated field subset (omit value for all fields)")
	cmd.Flag("json").NoOptDefVal = "*"
	flags.StringVarP(&f.JQ, "jq", "q", "", "filter JSON output through a jq expression")
	flags.StringVarP(&f.Template, "template", "t", "", "format JSON output using a Go template")
}

func (f *bareMsgFlags) args(cmd *cobra.Command, base []string) []string {
	out := append([]string(nil), base...)
	appendString := func(name, value string) {
		if cmd.Flags().Changed(name) {
			out = append(out, "--"+name+"="+value)
		}
	}
	appendBool := func(name string, value bool) {
		if cmd.Flags().Changed(name) {
			out = append(out, "--"+name+"="+strconv.FormatBool(value))
		}
	}
	appendString("in", f.In)
	appendString("filter", f.Filter)
	appendString("from", f.From)
	appendString("min-date", f.MinDate)
	appendString("max-date", f.MaxDate)
	appendBool("broadcasts-only", f.BroadcastsOnly)
	appendBool("groups-only", f.GroupsOnly)
	appendBool("users-only", f.UsersOnly)
	appendBool("missed", f.Missed)
	if cmd.Flags().Changed("limit") {
		out = append(out, "--limit="+strconv.Itoa(f.Limit))
	}
	appendString("order", f.Order)
	appendString("json", f.JSON)
	appendString("jq", f.JQ)
	appendString("template", f.Template)
	return out
}
