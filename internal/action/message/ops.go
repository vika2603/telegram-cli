package message

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// SendRequest is the raw request for `tg msg send`.
type SendRequest struct {
	RawRef   string
	Text     string
	Files    []string
	Names    []string
	ReplyTo  int
	Silent   bool
	Schedule time.Time
	Parse    string
	RandomID int64
	Stdin    io.Reader
}

// SendQuery is the normalized payload passed to Telegram.
type SendQuery struct {
	Ref         ref.Ref
	Text        string
	Attachments []Attachment
	ReplyTo     int
	Silent      bool
	Schedule    time.Time
	Parse       string
	// RandomID, when non-zero, is the message dedup key Telegram uses to
	// drop duplicate sends. Reusing the same value across retries makes a
	// resend idempotent. Zero means "let the client pick a fresh random
	// id" (the default, non-idempotent behavior).
	RandomID int64
	Stdin    io.Reader
}

// Attachment is one file sent by `tg msg send --file`.
type Attachment struct {
	Path string
	Name string
}

// SendFunc sends the normalized message payload.
type SendFunc func(context.Context, SendQuery) ([]output.SendResultRow, error)

// DownloadRequest is the raw request for `tg msg download`.
type DownloadRequest struct {
	RawMessageRef string
	Output        string
}

// DownloadQuery is the normalized payload passed to Telegram.
type DownloadQuery struct {
	Ref     ref.Ref
	Message int
	Output  string
}

// DownloadFunc downloads one message attachment.
type DownloadFunc func(context.Context, DownloadQuery) (output.DownloadRow, error)

// Send validates and dispatches a send request.
func Send(ctx context.Context, req SendRequest, do SendFunc) ([]output.SendResultRow, error) {
	query, err := NormalizeSend(req)
	if err != nil {
		return nil, err
	}
	if do == nil {
		return nil, fmt.Errorf("%w: msg send called without send function", command.ErrPrecondition)
	}
	return do(ctx, query)
}

// NormalizeSend resolves stdin-backed text and parses the peer ref.
func NormalizeSend(req SendRequest) (SendQuery, error) {
	switch req.Parse {
	case "", "html":
	case "markdown":
		return SendQuery{}, fmt.Errorf("%w: --parse markdown is not supported (gotd has no markdown parser); use --parse html or send plain text", command.ErrUsage)
	default:
		return SendQuery{}, fmt.Errorf("%w: unknown --parse value %q (supported: html)", command.ErrUsage, req.Parse)
	}
	files := compactStrings(req.Files)
	names := compactStrings(req.Names)
	if len(names) > 0 && len(files) == 0 {
		return SendQuery{}, fmt.Errorf("%w: --name requires --file", command.ErrUsage)
	}
	if len(names) > 0 && len(names) != len(files) {
		return SendQuery{}, fmt.Errorf("%w: repeat --name once for each --file", command.ErrUsage)
	}
	if req.Text == "-" && hasStdinFile(files) {
		return SendQuery{}, fmt.Errorf("%w: text and --file cannot both read from stdin", command.ErrUsage)
	}
	if err := validateStdinFiles(files); err != nil {
		return SendQuery{}, err
	}
	text, err := resolveText(req.Text, req.Stdin)
	if err != nil {
		return SendQuery{}, err
	}
	if text == "" && len(files) == 0 {
		return SendQuery{}, fmt.Errorf("%w: text or --file is required", command.ErrUsage)
	}
	attachments, err := normalizeAttachments(files, names)
	if err != nil {
		return SendQuery{}, err
	}
	// A single --random-id can't key an album: gotd assigns one random id
	// per media item, so there is no single dedup key to reuse on retry.
	if req.RandomID != 0 && len(attachments) > 1 {
		return SendQuery{}, fmt.Errorf("%w: --random-id is not supported with multiple attachments", command.ErrUsage)
	}
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return SendQuery{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	return SendQuery{
		Ref:         parsed,
		Text:        text,
		Attachments: attachments,
		ReplyTo:     req.ReplyTo,
		Silent:      req.Silent,
		Schedule:    req.Schedule,
		Parse:       req.Parse,
		RandomID:    req.RandomID,
		Stdin:       req.Stdin,
	}, nil
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func hasStdinFile(files []string) bool {
	for _, file := range files {
		if file == "-" {
			return true
		}
	}
	return false
}

func validateStdinFiles(files []string) error {
	stdinCount := 0
	for _, file := range files {
		if file == "-" {
			stdinCount++
		}
	}
	if stdinCount > 1 {
		return fmt.Errorf("%w: --file - can only be used once", command.ErrUsage)
	}
	if stdinCount == 1 && len(files) > 1 {
		return fmt.Errorf("%w: --file - cannot be combined with other files", command.ErrUsage)
	}
	return nil
}

func normalizeAttachments(files, names []string) ([]Attachment, error) {
	if len(files) == 0 {
		return nil, nil
	}
	out := make([]Attachment, 0, len(files))
	for i, file := range files {
		name := ""
		if len(names) > 0 {
			name = names[i]
			if name == "" {
				return nil, fmt.Errorf("%w: --name cannot be empty", command.ErrUsage)
			}
		}
		if file == "-" {
			if name == "" {
				name = "stdin"
			}
			out = append(out, Attachment{Path: file, Name: name})
			continue
		}
		if _, err := os.Stat(file); err != nil {
			return nil, fmt.Errorf("%w: cannot open --file: %s", command.ErrUsage, err.Error())
		}
		if name == "" {
			name = filepath.Base(file)
		}
		out = append(out, Attachment{Path: file, Name: name})
	}
	return out, nil
}

// Download validates and dispatches a media download request.
func Download(ctx context.Context, req DownloadRequest, do DownloadFunc) (output.DownloadRow, error) {
	query, err := NormalizeDownload(req)
	if err != nil {
		return output.DownloadRow{}, err
	}
	if do == nil {
		return output.DownloadRow{}, fmt.Errorf("%w: msg download called without download function", command.ErrPrecondition)
	}
	return do(ctx, query)
}

// NormalizeDownload parses the message ref and output path.
func NormalizeDownload(req DownloadRequest) (DownloadQuery, error) {
	msgRef, err := parseMessageRef(req.RawMessageRef)
	if err != nil {
		return DownloadQuery{}, err
	}
	return DownloadQuery{
		Ref:     msgRef.Peer,
		Message: msgRef.MessageID,
		Output:  strings.TrimSpace(req.Output),
	}, nil
}

func resolveText(text string, stdin io.Reader) (string, error) {
	if text == "" {
		return "", nil
	}
	if text != "-" {
		return text, nil
	}
	if stdin == nil {
		return "", fmt.Errorf("%w: text '-' requires stdin to be available", command.ErrUsage)
	}
	b, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("reading stdin: %w", err)
	}
	return strings.TrimRight(string(b), "\n"), nil
}

// EditRequest is the raw request for `tg msg edit`.
type EditRequest struct {
	RawMessageRef string
	Text          string
	Parse         string
}

// EditQuery is the normalized payload passed to Telegram.
type EditQuery struct {
	Ref       ref.Ref
	MessageID int
	Text      string
	Parse     string
}

// EditFunc edits a message.
type EditFunc func(context.Context, EditQuery) (output.SendResultRow, error)

// Edit validates and dispatches an edit request.
func Edit(ctx context.Context, req EditRequest, do EditFunc) (output.SendResultRow, error) {
	query, err := NormalizeEdit(req)
	if err != nil {
		return output.SendResultRow{}, err
	}
	if do == nil {
		return output.SendResultRow{}, fmt.Errorf("%w: msg edit called without edit function", command.ErrPrecondition)
	}
	return do(ctx, query)
}

// NormalizeEdit parses edit flags and refs.
func NormalizeEdit(req EditRequest) (EditQuery, error) {
	if req.Text == "" {
		return EditQuery{}, fmt.Errorf("%w: --text is required", command.ErrUsage)
	}
	switch req.Parse {
	case "", "html":
	case "markdown":
		return EditQuery{}, fmt.Errorf("%w: --parse markdown is not supported (gotd has no markdown parser); use --parse html or plain text", command.ErrUsage)
	default:
		return EditQuery{}, fmt.Errorf("%w: unknown --parse value %q (supported: html)", command.ErrUsage, req.Parse)
	}
	msgRef, err := parseMessageRef(req.RawMessageRef)
	if err != nil {
		return EditQuery{}, err
	}
	return EditQuery{Ref: msgRef.Peer, MessageID: msgRef.MessageID, Text: req.Text, Parse: req.Parse}, nil
}

// ForwardRequest is the raw request for `tg msg forward`.
type ForwardRequest struct {
	RawMessageRefs []string
	RawTo          string
	Silent         bool
	RandomID       int64
}

// ForwardQuery is the normalized payload passed to Telegram.
type ForwardQuery struct {
	From   ref.Ref
	To     ref.Ref
	IDs    []int
	Silent bool
	// RandomID is the dedup key (see SendQuery.RandomID). forwardMessages
	// assigns one id per forwarded message, so a single key only maps to a
	// single-message forward; NormalizeForward rejects it for multiple.
	RandomID int64
}

// ForwardFunc forwards messages.
type ForwardFunc func(context.Context, ForwardQuery) (output.SendResultRow, error)

// Forward validates and dispatches a forward request.
func Forward(ctx context.Context, req ForwardRequest, do ForwardFunc) (output.SendResultRow, error) {
	query, err := NormalizeForward(req)
	if err != nil {
		return output.SendResultRow{}, err
	}
	if do == nil {
		return output.SendResultRow{}, fmt.Errorf("%w: msg forward called without forward function", command.ErrPrecondition)
	}
	return do(ctx, query)
}

// NormalizeForward parses refs and validates destination.
func NormalizeForward(req ForwardRequest) (ForwardQuery, error) {
	if req.RawTo == "" {
		return ForwardQuery{}, fmt.Errorf("%w: --to is required", command.ErrUsage)
	}
	from, ids, err := parseMessageRefs(req.RawMessageRefs)
	if err != nil {
		return ForwardQuery{}, err
	}
	to, err := ref.ParseRef(req.RawTo)
	if err != nil {
		return ForwardQuery{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	// forwardMessages needs one distinct random_id per message, so a single
	// --random-id can only key a single-message forward (same constraint as
	// an album in NormalizeSend).
	if req.RandomID != 0 && len(ids) > 1 {
		return ForwardQuery{}, fmt.Errorf("%w: --random-id is not supported when forwarding multiple messages", command.ErrUsage)
	}
	return ForwardQuery{From: from, To: to, IDs: ids, Silent: req.Silent, RandomID: req.RandomID}, nil
}

// DeleteRequest is the raw request for `tg msg delete`.
type DeleteRequest struct {
	RawMessageRefs []string
	Revoke         bool
	Yes            bool
	Prompter       ui.Prompter
}

// DeleteQuery is the normalized payload passed to Telegram.
type DeleteQuery struct {
	Ref    ref.Ref
	IDs    []int
	Revoke bool
}

// DeleteResult describes deleted message output.
type DeleteResult struct {
	Verb  string
	Count int
}

// DeleteFunc deletes messages.
type DeleteFunc func(context.Context, DeleteQuery) error

// Delete validates, confirms, and dispatches a delete request.
func Delete(ctx context.Context, req DeleteRequest, do DeleteFunc) (DeleteResult, error) {
	query, err := NormalizeDelete(req)
	if err != nil {
		return DeleteResult{}, err
	}
	if do == nil {
		return DeleteResult{}, fmt.Errorf("%w: msg delete called without delete function", command.ErrPrecondition)
	}
	verb := "delete"
	if query.Revoke {
		verb = "revoke for everyone"
	}
	if err := ui.ConfirmDestructive(req.Prompter, fmt.Sprintf("%s %d message(s) in %s?", verb, len(query.IDs), query.Ref.String()), req.Yes); err != nil {
		return DeleteResult{}, err
	}
	if err := do(ctx, query); err != nil {
		return DeleteResult{}, err
	}
	outVerb := "deleted"
	if query.Revoke {
		outVerb = "revoked"
	}
	return DeleteResult{Verb: outVerb, Count: len(query.IDs)}, nil
}

// NormalizeDelete parses delete refs.
func NormalizeDelete(req DeleteRequest) (DeleteQuery, error) {
	parsed, ids, err := parseMessageRefs(req.RawMessageRefs)
	if err != nil {
		return DeleteQuery{}, err
	}
	return DeleteQuery{Ref: parsed, IDs: ids, Revoke: req.Revoke}, nil
}

// PinRequest is the raw request for `tg msg pin` and `tg msg unpin`.
type PinRequest struct {
	RawMessageRef string
	Silent        bool
	Unpin         bool
}

// PinQuery is the normalized payload passed to Telegram.
type PinQuery struct {
	Ref       ref.Ref
	MessageID int
	Silent    bool
	Unpin     bool
}

// PinResult describes pin output.
type PinResult struct {
	Verb      string
	MessageID int
}

// PinFunc updates pin state.
type PinFunc func(context.Context, PinQuery) error

// Pin validates and dispatches a pin request.
func Pin(ctx context.Context, req PinRequest, do PinFunc) (PinResult, error) {
	query, err := NormalizePin(req)
	if err != nil {
		return PinResult{}, err
	}
	if do == nil {
		return PinResult{}, fmt.Errorf("%w: msg pin called without pin function", command.ErrPrecondition)
	}
	if err := do(ctx, query); err != nil {
		return PinResult{}, err
	}
	verb := "pinned"
	if query.Unpin {
		verb = "unpinned"
	}
	return PinResult{Verb: verb, MessageID: query.MessageID}, nil
}

// NormalizePin parses the pin peer ref.
func NormalizePin(req PinRequest) (PinQuery, error) {
	msgRef, err := parseMessageRef(req.RawMessageRef)
	if err != nil {
		return PinQuery{}, err
	}
	return PinQuery{Ref: msgRef.Peer, MessageID: msgRef.MessageID, Silent: req.Silent, Unpin: req.Unpin}, nil
}

// ReactRequest is the raw request for `tg msg react`.
type ReactRequest struct {
	RawMessageRef string
	Emoji         string
	Clear         bool
}

// ReactQuery is the normalized payload passed to Telegram.
type ReactQuery struct {
	Ref       ref.Ref
	MessageID int
	Emoji     string
	Clear     bool
}

// ReactFunc updates a reaction.
type ReactFunc func(context.Context, ReactQuery) (output.SendResultRow, error)

// React validates and dispatches a reaction request.
func React(ctx context.Context, req ReactRequest, do ReactFunc) (output.SendResultRow, error) {
	query, err := NormalizeReact(req)
	if err != nil {
		return output.SendResultRow{}, err
	}
	if do == nil {
		return output.SendResultRow{}, fmt.Errorf("%w: msg react called without react function", command.ErrPrecondition)
	}
	return do(ctx, query)
}

// NormalizeReact parses refs and validates reaction flags.
func NormalizeReact(req ReactRequest) (ReactQuery, error) {
	if !req.Clear && req.Emoji == "" {
		return ReactQuery{}, fmt.Errorf("%w: exactly one of --emoji or --clear must be provided", command.ErrUsage)
	}
	if req.Clear && req.Emoji != "" {
		return ReactQuery{}, fmt.Errorf("%w: --emoji and --clear are mutually exclusive", command.ErrUsage)
	}
	msgRef, err := parseMessageRef(req.RawMessageRef)
	if err != nil {
		return ReactQuery{}, err
	}
	return ReactQuery{Ref: msgRef.Peer, MessageID: msgRef.MessageID, Emoji: req.Emoji, Clear: req.Clear}, nil
}

// CancelScheduledRequest is the raw request for `tg msg schedule-cancel`.
type CancelScheduledRequest struct {
	RawRef   string
	IDs      []int
	Yes      bool
	Prompter ui.Prompter
}

// CancelScheduledQuery is the normalized payload passed to Telegram.
type CancelScheduledQuery struct {
	Ref ref.Ref
	IDs []int
}

// CancelScheduledResult describes cancel output.
type CancelScheduledResult struct {
	Count int
}

// CancelScheduledFunc cancels scheduled messages.
type CancelScheduledFunc func(context.Context, CancelScheduledQuery) error

// CancelScheduled validates, confirms, and dispatches a cancel request.
func CancelScheduled(ctx context.Context, req CancelScheduledRequest, do CancelScheduledFunc) (CancelScheduledResult, error) {
	query, err := NormalizeCancelScheduled(req)
	if err != nil {
		return CancelScheduledResult{}, err
	}
	if do == nil {
		return CancelScheduledResult{}, fmt.Errorf("%w: msg schedule-cancel called without cancel function", command.ErrPrecondition)
	}
	if err := ui.ConfirmDestructive(req.Prompter, fmt.Sprintf("cancel %d scheduled message(s) in %s?", len(query.IDs), req.RawRef), req.Yes); err != nil {
		return CancelScheduledResult{}, err
	}
	if err := do(ctx, query); err != nil {
		return CancelScheduledResult{}, err
	}
	return CancelScheduledResult{Count: len(query.IDs)}, nil
}

// NormalizeCancelScheduled parses the scheduled message peer ref.
func NormalizeCancelScheduled(req CancelScheduledRequest) (CancelScheduledQuery, error) {
	if err := validateMessageIDs(req.IDs); err != nil {
		return CancelScheduledQuery{}, err
	}
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return CancelScheduledQuery{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	return CancelScheduledQuery{Ref: parsed, IDs: req.IDs}, nil
}

// ScheduledListRequest is the raw request for `tg msg schedule-list`.
type ScheduledListRequest struct {
	RawRef string
}

// ScheduledListQuery is the normalized payload passed to Telegram.
type ScheduledListQuery struct {
	Ref ref.Ref
}

// ScheduledListFunc loads scheduled message rows.
type ScheduledListFunc func(context.Context, ScheduledListQuery) ([]output.ScheduledMessageRow, error)

// ScheduledList validates and dispatches a scheduled list request.
func ScheduledList(ctx context.Context, req ScheduledListRequest, fetch ScheduledListFunc) ([]output.ScheduledMessageRow, error) {
	query, err := NormalizeScheduledList(req)
	if err != nil {
		return nil, err
	}
	if fetch == nil {
		return nil, fmt.Errorf("%w: msg schedule-list called without fetch function", command.ErrPrecondition)
	}
	return fetch(ctx, query)
}

// NormalizeScheduledList parses the peer ref.
func NormalizeScheduledList(req ScheduledListRequest) (ScheduledListQuery, error) {
	parsed, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return ScheduledListQuery{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	return ScheduledListQuery{Ref: parsed}, nil
}

// LinkRequest is the raw request for `tg msg link`.
type LinkRequest struct {
	RawMessageRef string
}

// LinkQuery is the normalized payload passed to Telegram.
type LinkQuery struct {
	Ref       ref.Ref
	MessageID int
}

// LinkPeer carries the minimal peer information needed to build a t.me link.
type LinkPeer struct {
	Username  string
	ChannelID int64
	IsChannel bool
}

// LinkResolveFunc resolves link peer metadata.
type LinkResolveFunc func(context.Context, LinkQuery) (LinkPeer, error)

// Link validates, resolves, and builds a message link.
func Link(ctx context.Context, req LinkRequest, resolve LinkResolveFunc) (string, error) {
	query, err := NormalizeLink(req)
	if err != nil {
		return "", err
	}
	if resolve == nil {
		return "", fmt.Errorf("%w: msg link called without resolve function", command.ErrPrecondition)
	}
	p, err := resolve(ctx, query)
	if err != nil {
		return "", err
	}
	if !p.IsChannel {
		return "", fmt.Errorf("%w: t.me links are only emitted for channel/supergroup messages", ErrNoLink)
	}
	if p.Username != "" {
		return fmt.Sprintf("https://t.me/%s/%d", p.Username, query.MessageID), nil
	}
	return fmt.Sprintf("https://t.me/c/%d/%d", p.ChannelID, query.MessageID), nil
}

// NormalizeLink parses link refs.
func NormalizeLink(req LinkRequest) (LinkQuery, error) {
	msgRef, err := parseMessageRef(req.RawMessageRef)
	if err != nil {
		return LinkQuery{}, err
	}
	return LinkQuery{Ref: msgRef.Peer, MessageID: msgRef.MessageID}, nil
}

func parseMessageRef(raw string) (ref.MessageRef, error) {
	parsed, err := ref.ParseMessageRef(raw)
	if err != nil {
		return ref.MessageRef{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	return parsed, nil
}

func parseMessageRefs(raws []string) (ref.Ref, []int, error) {
	if len(raws) == 0 {
		return ref.Ref{}, nil, fmt.Errorf("%w: at least one message ref is required", command.ErrUsage)
	}
	first, err := parseMessageRef(raws[0])
	if err != nil {
		return ref.Ref{}, nil, err
	}
	ids := make([]int, 0, len(raws))
	ids = append(ids, first.MessageID)
	for _, raw := range raws[1:] {
		next, err := parseMessageRef(raw)
		if err != nil {
			return ref.Ref{}, nil, err
		}
		if next.Peer.String() != first.Peer.String() {
			return ref.Ref{}, nil, fmt.Errorf("%w: message refs must belong to the same peer", command.ErrUsage)
		}
		ids = append(ids, next.MessageID)
	}
	return first.Peer, ids, nil
}

func validateMessageIDs(ids []int) error {
	if len(ids) == 0 {
		return fmt.Errorf("%w: at least one message id is required", command.ErrUsage)
	}
	for _, id := range ids {
		if id <= 0 {
			return fmt.Errorf("%w: message id must be positive", command.ErrUsage)
		}
	}
	return nil
}
