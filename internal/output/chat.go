package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gotd/td/tg"

	"github.com/vika2603/telegram-cli/internal/ref"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// ChatRow is the output of `chat list`, `chat show`, and `search chat`.
// "Kind" is one of: "user", "bot", "chat" (small group), "channel"
// (broadcast).
type ChatRow struct {
	ID          int64           `json:"id"`
	Ref         string          `json:"ref,omitempty"`
	Kind        string          `json:"kind"`
	Title       string          `json:"title,omitempty"`
	Username    string          `json:"username,omitempty"`
	UnreadCount int             `json:"unread_count,omitempty"`
	IsPinned    bool            `json:"is_pinned,omitempty"`
	IsArchived  bool            `json:"is_archived,omitempty"`
	IsMuted     bool            `json:"is_muted,omitempty"`
	TopMessage  int             `json:"top_message,omitempty"`
	Last        *MessageSummary `json:"last,omitempty"`
}

func (r ChatRow) MarshalJSON() ([]byte, error) {
	type chatRowJSON struct {
		Peer       PeerObject      `json:"peer"`
		Title      string          `json:"title,omitempty"`
		Type       string          `json:"type,omitempty"`
		Unread     int             `json:"unread,omitempty"`
		Pinned     bool            `json:"pinned,omitempty"`
		Archived   bool            `json:"archived,omitempty"`
		Muted      bool            `json:"muted,omitempty"`
		TopMessage int             `json:"top_message,omitempty"`
		Last       *MessageSummary `json:"last,omitempty"`
	}
	return json.Marshal(chatRowJSON{
		Peer:       peerObject(r.Ref, r.ID, r.Kind, r.Title, r.Username),
		Title:      r.Title,
		Type:       r.Kind,
		Unread:     r.UnreadCount,
		Pinned:     r.IsPinned,
		Archived:   r.IsArchived,
		Muted:      r.IsMuted,
		TopMessage: r.TopMessage,
		Last:       r.Last,
	})
}

func RenderChatList(io *ui.IOStreams, rows []ChatRow) error {
	if io.IsStdoutTTY() {
		return renderChatListTTY(io, rows)
	}
	tp := NewTablePrinter(io)
	tp.AddHeader("REF", "KIND", "TITLE", "USERNAME", "UNREAD", "LAST")
	for _, r := range rows {
		tp.AddRow(
			displayRef(r),
			r.Kind,
			r.Title,
			prefixUsername(r.Username),
			itoaOrBlank(r.UnreadCount),
			shortText(lastPreview(r.Last), 48),
		)
	}
	return tp.Render()
}

func renderChatListTTY(io *ui.IOStreams, rows []ChatRow) error {
	width := io.TerminalWidth()
	if width <= 0 {
		width = 80
	}
	colors := io.ColorScheme()
	refWidth := chatListRefWidth(rows)
	metaWidth := chatListMetaWidth(rows)
	titleWidth := min(width-refWidth-metaWidth-4, 36)
	if titleWidth < 16 {
		titleWidth = 16
	}
	previewWidth := min(width-refWidth-2, 56)
	if previewWidth < 16 {
		previewWidth = min(width-2, 56)
	}
	previewIndent := strings.Repeat(" ", min(refWidth+2, width-previewWidth))
	for _, r := range rows {
		ref := displayRef(r)
		meta := chatMeta(r)
		stacked := displayWidth(ref) > refWidth || refWidth+metaWidth+22 > width
		if stacked {
			if _, err := fmt.Fprintln(io.Out, colors.Bold(ref)); err != nil {
				return err
			}
			titleLine := displayChatTitle(r)
			if meta != "" {
				titleLine = fitText(titleLine, max(width-displayWidth(meta)-4, 16)) + "  " + colors.Gray(meta)
			}
			if titleLine != "" {
				if _, err := fmt.Fprintf(io.Out, "  %s\n", fitText(titleLine, width-2)); err != nil {
					return err
				}
			}
			if last := shortText(lastPreview(r.Last), min(width-2, 72)); last != "" {
				if _, err := fmt.Fprintf(io.Out, "  %s\n", colors.Gray(last)); err != nil {
					return err
				}
			}
			continue
		}
		line := colors.Bold(padRight(fitText(ref, refWidth), refWidth)) +
			"  " +
			fitText(displayChatTitle(r), titleWidth) +
			"  " +
			colors.Gray(fitText(meta, metaWidth))
		if _, err := fmt.Fprintln(io.Out, fitText(line, width)); err != nil {
			return err
		}
		if last := shortText(lastPreview(r.Last), previewWidth); last != "" {
			if _, err := fmt.Fprintf(io.Out, "%s%s\n", previewIndent, colors.Gray(last)); err != nil {
				return err
			}
		}
	}
	return nil
}

func chatListRefWidth(rows []ChatRow) int {
	width := 0
	for _, r := range rows {
		width = max(width, displayWidth(displayRef(r)))
	}
	return min(max(width, 12), 22)
}

func chatListMetaWidth(rows []ChatRow) int {
	width := 0
	for _, r := range rows {
		width = max(width, displayWidth(chatMeta(r)))
	}
	return min(max(width, 6), 22)
}

func chatMeta(r ChatRow) string {
	parts := []string{r.Kind}
	if r.UnreadCount > 0 {
		parts = append(parts, strconv.Itoa(r.UnreadCount)+" unread")
	}
	return strings.Join(parts, " · ")
}

func displayChatTitle(r ChatRow) string {
	username := prefixUsername(r.Username)
	if username == displayRef(r) {
		username = ""
	}
	switch {
	case r.Title != "" && username != "":
		return r.Title + " " + username
	case r.Title != "":
		return r.Title
	case username != "":
		return username
	default:
		return displayRef(r)
	}
}

func lastPreview(last *MessageSummary) string {
	if last == nil {
		return ""
	}
	if last.Text != "" {
		return last.Text
	}
	if last.Media != nil && last.Media.Type != "" {
		return "[" + last.Media.Type + "]"
	}
	return ""
}

// RenderChatShow prints a single-chat detail block.
func RenderChatShow(io *ui.IOStreams, r ChatRow) error {
	tp := NewTablePrinter(io)
	if r.Ref != "" {
		tp.AddRow("REF", r.Ref)
	}
	tp.AddRow("ID", strconv.FormatInt(r.ID, 10))
	tp.AddRow("KIND", r.Kind)
	if r.Title != "" {
		tp.AddRow("TITLE", r.Title)
	}
	if r.Username != "" {
		tp.AddRow("USERNAME", "@"+r.Username)
	}
	if r.UnreadCount > 0 {
		tp.AddRow("UNREAD", strconv.Itoa(r.UnreadCount))
	}
	if r.IsPinned {
		tp.AddRow("PINNED", "true")
	}
	if r.IsArchived {
		tp.AddRow("ARCHIVED", "true")
	}
	if r.IsMuted {
		tp.AddRow("MUTED", "true")
	}
	return tp.Render()
}

func prefixUsername(u string) string {
	if u == "" {
		return ""
	}
	return "@" + u
}

func displayRef(r ChatRow) string {
	if r.Ref != "" {
		return r.Ref
	}
	return strconv.FormatInt(r.ID, 10)
}

// PeerRef is a compact peer identity nested in mutation rows (join/leave,
// mute/unmute, archive/unarchive, etc.). Intentionally smaller than
// ChatRow — no unread/pinned/archived state, only "who is this peer".
type PeerRef struct {
	ID       int64  `json:"id"`
	Ref      string `json:"ref,omitempty"`
	Kind     string `json:"kind"` // "user" | "chat" | "channel"
	Title    string `json:"title,omitempty"`
	Username string `json:"username,omitempty"`
}

// ChatMembershipRow is emitted by `chat join` and `chat leave`.
// AlreadyMember distinguishes "no-op, you were already in" from a fresh
// transition. Role is only set by join (Telegram returns it on JoinChannel).
type ChatMembershipRow struct {
	Action        string  `json:"action"` // "join" | "leave"
	Peer          PeerRef `json:"peer"`
	AlreadyMember bool    `json:"already_member,omitempty"`
	Role          string  `json:"role,omitempty"` // "member" | "admin" | "creator"
}

// PeerRefFromResolved converts a peer.Resolved into a compact PeerRef.
func PeerRefFromResolved(r peer.Resolved) PeerRef {
	return PeerRef{
		ID:       r.ID,
		Ref:      PreferredRefFromResolved(r),
		Kind:     r.Kind,
		Title:    r.Title,
		Username: r.Username,
	}
}

// PeerRefFromChat maps a tg.ChatClass into a PeerRef. Empty result for
// unrecognized concrete types (shouldn't happen in practice but stays safe).
func PeerRefFromChat(c tg.ChatClass) PeerRef {
	switch v := c.(type) {
	case *tg.Chat:
		return PeerRef{ID: -v.ID, Ref: ref.FormatPeer("chat", v.ID, 0), Kind: "chat", Title: v.Title}
	case *tg.Channel:
		kind := "channel"
		if !v.Broadcast {
			kind = "chat"
		}
		return PeerRef{ID: -1_000_000_000_000 - v.ID, Ref: preferredPeerRef(kind, v.Username, v.ID, v.AccessHash, true), Kind: kind, Title: v.Title, Username: v.Username}
	}
	return PeerRef{}
}

// PreferredRefFromResolved returns the shortest command input that can resolve
// the peer again.
func PreferredRefFromResolved(r peer.Resolved) string {
	if r.Username != "" {
		return "@" + r.Username
	}
	switch p := r.InputPeer.(type) {
	case *tg.InputPeerUser:
		return ref.FormatPeer("user", p.UserID, p.AccessHash)
	case *tg.InputPeerChat:
		return ref.FormatPeer("chat", p.ChatID, 0)
	case *tg.InputPeerChannel:
		return ref.FormatPeer("channel", p.ChannelID, p.AccessHash)
	case *tg.InputPeerSelf:
		return "me"
	default:
		return ""
	}
}

// PreferredPeerRef builds the shortest command input for a row when the caller
// has raw Telegram peer fields rather than a peer.Resolved value.
func PreferredPeerRef(kind, username string, rawID, accessHash int64, inputIsChannel bool) string {
	return preferredPeerRef(kind, username, rawID, accessHash, inputIsChannel)
}

func preferredPeerRef(kind, username string, rawID, accessHash int64, inputIsChannel bool) string {
	if username != "" {
		return "@" + username
	}
	if inputIsChannel || kind == "channel" {
		return ref.FormatPeer("channel", rawID, accessHash)
	}
	if kind == "chat" {
		return ref.FormatPeer("chat", rawID, 0)
	}
	return ref.FormatPeer("user", rawID, accessHash)
}

// WriteChatMembershipJSON emits one ndjson line.
func WriteChatMembershipJSON(w io.Writer, r ChatMembershipRow) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}

// ChatMuteRow is emitted by `chat mute` / `chat unmute`. MuteUntil is absent
// on unmute (which sets the underlying field to 0).
type ChatMuteRow struct {
	Action    string  `json:"action"` // "mute" | "unmute"
	Peer      PeerRef `json:"peer"`
	MuteUntil string  `json:"mute_until,omitempty"` // RFC3339; "forever" for MaxInt32
}

// WriteChatMuteJSON emits one ndjson line.
func WriteChatMuteJSON(w io.Writer, r ChatMuteRow) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}

// ChatPinRow is emitted by `chat pin` / `chat unpin`. It reports whether the
// dialog is pinned to the top of the chat list after the call.
type ChatPinRow struct {
	Action string  `json:"action"` // "pin" | "unpin"
	Peer   PeerRef `json:"peer"`
	Pinned bool    `json:"pinned"`
}

// WriteChatPinJSON emits one ndjson line.
func WriteChatPinJSON(w io.Writer, r ChatPinRow) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}

// ChatFolderRow is emitted by `chat archive` / `chat unarchive`.
// Folder is the numeric folder ID (0 = main, 1 = archive).
type ChatFolderRow struct {
	Action string  `json:"action"` // "archive" | "unarchive"
	Peer   PeerRef `json:"peer"`
	Folder int     `json:"folder"`
}

// WriteChatFolderJSON emits one ndjson line.
func WriteChatFolderJSON(w io.Writer, r ChatFolderRow) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}
