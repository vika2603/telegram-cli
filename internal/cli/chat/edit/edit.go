// Package edit implements "tg chat edit <ref>".
package edit

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/cli/complete"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Scope controls which scope-specific flags are registered on the command.
type Scope int

const (
	// ScopeChat registers supergroup-specific flags (forum, hide-members, hide-history, slow-mode).
	ScopeChat Scope = iota
	// ScopeChannel registers channel-specific flags (signatures).
	ScopeChannel
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	RawRef      string
	Title       *string
	About       *string
	Username    *string
	Forum       *bool
	HideMembers *bool
	HideHistory *bool
	SlowMode    *int
	NoForwards  *bool
	Signatures  *bool
	JoinRequest *bool
	Exporter    output.Exporter
	IOStreams   *ui.IOStreams
	Edit        actionchat.EditChatFunc
}

// New builds the cobra command for "tg chat edit". It is shared by
// "tg channel edit" via NewWith.
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	return NewWith(f, runF, "Edit a supergroup: title, about, visibility, and settings", ScopeChat)
}

// NewWith builds the edit command with a caller-supplied short description and
// scope so the channel tree can reuse it with channel-specific help and flags.
func NewWith(f *runtime.Invocation, runF func(*Options) error, short string, scope Scope) *cobra.Command {
	opts := &Options{}
	var title, about, public string
	var private bool
	// tri-state pairs
	var forum, noForum bool
	var hideMembers, showMembers bool
	var hideHistory, showHistory bool
	var noForwards, allowForwards bool
	var signatures, noSignatures bool
	var joinRequest, noJoinRequest bool
	var slowMode int

	cmd := &cobra.Command{
		Use:               "edit <ref>",
		Short:             short,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams

			if cmd.Flags().Changed("title") {
				opts.Title = &title
			}
			if cmd.Flags().Changed("about") {
				opts.About = &about
			}

			// --public and --private are mutually exclusive; --public carries the
			// username string, --private sets an empty string (make private).
			publicSet := cmd.Flags().Changed("public")
			if publicSet && private {
				return fmt.Errorf("%w: --public and --private are mutually exclusive", command.ErrUsage)
			}
			switch {
			case publicSet:
				opts.Username = &public
			case private:
				empty := ""
				opts.Username = &empty
			}

			var err error
			opts.NoForwards, err = triBool(cmd, "no-forwards", "allow-forwards")
			if err != nil {
				return err
			}

			if scope == ScopeChat {
				opts.Forum, err = triBool(cmd, "forum", "no-forum")
				if err != nil {
					return err
				}
				opts.HideMembers, err = triBool(cmd, "hide-members", "show-members")
				if err != nil {
					return err
				}
				opts.HideHistory, err = triBool(cmd, "hide-history", "show-history")
				if err != nil {
					return err
				}
				if cmd.Flags().Changed("slow-mode") {
					opts.SlowMode = &slowMode
				}
				opts.JoinRequest, err = triBool(cmd, "need-approval", "no-need-approval")
				if err != nil {
					return err
				}
			}

			if scope == ScopeChannel {
				opts.Signatures, err = triBool(cmd, "signatures", "no-signatures")
				if err != nil {
					return err
				}
			}

			if runF != nil {
				return runF(opts)
			}
			opts.Edit = newEditFn(f)
			return Run(cmd.Context(), opts)
		},
	}

	// Common flags (both scopes).
	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&about, "about", "", "New description / about text")
	cmd.Flags().StringVar(&public, "public", "", "Make public with this @username")
	cmd.Flags().BoolVar(&private, "private", false, "Make private (remove the public username, invite-only)")
	cmd.Flags().BoolVar(&noForwards, "no-forwards", false, "Forbid forwarding and saving content")
	cmd.Flags().BoolVar(&allowForwards, "allow-forwards", false, "Allow forwarding and saving content")

	// ScopeChat-only flags.
	if scope == ScopeChat {
		cmd.Flags().BoolVar(&forum, "forum", false, "Enable topics (forum mode)")
		cmd.Flags().BoolVar(&noForum, "no-forum", false, "Disable topics (forum mode)")
		cmd.Flags().BoolVar(&hideMembers, "hide-members", false, "Hide member list from non-admins")
		cmd.Flags().BoolVar(&showMembers, "show-members", false, "Show member list to all members")
		cmd.Flags().BoolVar(&hideHistory, "hide-history", false, "Hide chat history from new members")
		cmd.Flags().BoolVar(&showHistory, "show-history", false, "Show chat history to new members")
		cmd.Flags().IntVar(&slowMode, "slow-mode", 0, "Slow mode delay in seconds (0 = off)")
		cmd.Flags().BoolVar(&joinRequest, "need-approval", false, "Require admin approval to join (public groups only)")
		cmd.Flags().BoolVar(&noJoinRequest, "no-need-approval", false, "Allow joining without approval")
	}

	// ScopeChannel-only flags.
	if scope == ScopeChannel {
		cmd.Flags().BoolVar(&signatures, "signatures", false, "Show author signatures on posts")
		cmd.Flags().BoolVar(&noSignatures, "no-signatures", false, "Hide author signatures on posts")
	}

	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"peer", "title", "type"})
	return cmd
}

// triBool resolves a mutually exclusive positive/negative flag pair into a
// *bool: true if pos was set, false if neg was set, nil if neither, error if both.
func triBool(cmd *cobra.Command, pos, neg string) (*bool, error) {
	posSet := cmd.Flags().Changed(pos)
	negSet := cmd.Flags().Changed(neg)
	if posSet && negSet {
		return nil, fmt.Errorf("%w: --%s and --%s are mutually exclusive", command.ErrUsage, pos, neg)
	}
	switch {
	case posSet:
		v := true
		return &v, nil
	case negSet:
		v := false
		return &v, nil
	default:
		return nil, nil
	}
}

// Run dispatches the edit request and renders the updated chat.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionchat.EditChat(ctx, actionchat.EditChatRequest{
		RawRef:      opts.RawRef,
		Title:       opts.Title,
		About:       opts.About,
		Username:    opts.Username,
		Forum:       opts.Forum,
		HideMembers: opts.HideMembers,
		HideHistory: opts.HideHistory,
		SlowMode:    opts.SlowMode,
		NoForwards:  opts.NoForwards,
		Signatures:  opts.Signatures,
		JoinRequest: opts.JoinRequest,
	}, opts.Edit)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderChatShow(opts.IOStreams, row)
}

func newEditFn(f *runtime.Invocation) actionchat.EditChatFunc {
	return func(ctx context.Context, q actionchat.EditChatQuery) (output.ChatRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.ChatRow{}, err
		}
		var row output.ChatRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.EditChat(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
