// Package ref parses peer references from user input.
// It is pure: no gotd imports, no network, no state. Resolving a Ref to a
// Telegram InputPeer is the client's job.
package ref

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type RefKind int

const (
	RefKindInvalid RefKind = iota
	RefKindMe
	RefKindSaved
	RefKindUsername
	RefKindPhone
	RefKindID
	RefKindPeer
	RefKindTMeLink
	RefKindTGDeeplink
)

// Ref is a parsed peer reference. Exactly one of Value or ID is populated,
// depending on Kind.
type Ref struct {
	Kind       RefKind
	Value      string // username (without @), phone digits (without +), t.me path, tg:// body, or peer type
	ID         int64  // for RefKindID and RefKindPeer
	AccessHash int64  // for RefKindPeer when needed by Telegram
}

func (r Ref) String() string {
	switch r.Kind {
	case RefKindInvalid:
		return "<invalid>"
	case RefKindMe:
		return "me"
	case RefKindSaved:
		return "saved"
	case RefKindUsername:
		return "@" + r.Value
	case RefKindPhone:
		return "+" + r.Value
	case RefKindID:
		return strconv.FormatInt(r.ID, 10)
	case RefKindPeer:
		return FormatPeer(r.Value, r.ID, r.AccessHash)
	case RefKindTMeLink:
		return "t.me/" + r.Value
	case RefKindTGDeeplink:
		return "tg://" + r.Value
	}
	return "<invalid>"
}

func ParseRef(s string) (Ref, error) {
	if s == "" {
		return Ref{}, errors.New("empty peer reference")
	}
	if v, ok := stripTMePrefix(s); ok {
		if v == "" {
			return Ref{}, errors.New("t.me link must have a path")
		}
		return Ref{Kind: RefKindTMeLink, Value: v}, nil
	}
	if strings.HasPrefix(s, "tg://") {
		v := s[len("tg://"):]
		if v == "" {
			return Ref{}, errors.New("tg:// deeplink must have a body")
		}
		return Ref{Kind: RefKindTGDeeplink, Value: v}, nil
	}
	if r, ok, err := parsePeerRef(s); ok || err != nil {
		return r, err
	}
	switch s {
	case "me":
		return Ref{Kind: RefKindMe}, nil
	case "saved":
		return Ref{Kind: RefKindSaved}, nil
	}
	if s[0] == '@' {
		if len(s) < 2 {
			return Ref{}, errors.New("username must not be bare @")
		}
		name := s[1:]
		for _, c := range name {
			if !isUsernameChar(c) {
				return Ref{}, fmt.Errorf("invalid username char %q", c)
			}
		}
		return Ref{Kind: RefKindUsername, Value: name}, nil
	}
	if s[0] == '+' {
		digits := s[1:]
		if len(digits) == 0 {
			return Ref{}, errors.New("phone must have digits")
		}
		for _, c := range digits {
			if c < '0' || c > '9' {
				return Ref{}, fmt.Errorf("invalid phone char %q", c)
			}
		}
		return Ref{Kind: RefKindPhone, Value: digits}, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return Ref{}, fmt.Errorf("peer reference %q: not a username, phone, id, or reserved word", s)
	}
	return Ref{Kind: RefKindID, ID: n}, nil
}

// FormatPeer returns a compact, copy-pasteable ref for peers that cannot be
// addressed by username.
func FormatPeer(kind string, id, accessHash int64) string {
	switch kind {
	case "user":
		return fmt.Sprintf("u:%d:%d", id, accessHash)
	case "chat":
		return fmt.Sprintf("g:%d", id)
	case "channel":
		return fmt.Sprintf("c:%d:%d", id, accessHash)
	default:
		return strconv.FormatInt(id, 10)
	}
}

func parsePeerRef(s string) (Ref, bool, error) {
	parts := strings.Split(s, ":")
	if len(parts) < 2 {
		return Ref{}, false, nil
	}
	kind, ok := normalizePeerKind(parts[0])
	if !ok {
		return Ref{}, false, nil
	}
	switch kind {
	case "chat":
		if len(parts) != 2 {
			return Ref{}, true, errors.New("chat ref must be g:<id>")
		}
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || id <= 0 {
			return Ref{}, true, errors.New("chat ref id must be positive")
		}
		return Ref{Kind: RefKindPeer, Value: kind, ID: id}, true, nil
	case "user", "channel":
		if len(parts) != 3 {
			return Ref{}, true, fmt.Errorf("%s ref must be %s:<id>:<access_hash>", kind, parts[0])
		}
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || id <= 0 {
			return Ref{}, true, fmt.Errorf("%s ref id must be positive", kind)
		}
		hash, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return Ref{}, true, fmt.Errorf("%s ref access_hash must be an integer", kind)
		}
		return Ref{Kind: RefKindPeer, Value: kind, ID: id, AccessHash: hash}, true, nil
	default:
		return Ref{}, false, nil
	}
}

func normalizePeerKind(s string) (string, bool) {
	switch s {
	case "u", "user":
		return "user", true
	case "g", "group", "chat":
		return "chat", true
	case "c", "channel":
		return "channel", true
	default:
		return "", false
	}
}

// IsInviteLink reports whether r is a t.me invite link (t.me/+hash or
// t.me/joinchat/hash). Returns false for plain username t.me links.
func (r Ref) IsInviteLink() bool {
	return r.Kind == RefKindTMeLink &&
		(strings.HasPrefix(r.Value, "+") || strings.HasPrefix(r.Value, "joinchat/"))
}

// InviteHash returns the raw invite hash embedded in a t.me invite link.
// For t.me/+abc it returns "abc"; for t.me/joinchat/abc it also returns "abc".
// The result is undefined when IsInviteLink() is false.
func (r Ref) InviteHash() string {
	return strings.TrimPrefix(strings.TrimPrefix(r.Value, "joinchat/"), "+")
}

func isUsernameChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}

func stripTMePrefix(s string) (string, bool) {
	for _, p := range []string{"https://t.me/", "http://t.me/", "t.me/"} {
		if strings.HasPrefix(s, p) {
			return s[len(p):], true
		}
	}
	return "", false
}
