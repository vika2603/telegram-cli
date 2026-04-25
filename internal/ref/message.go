package ref

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// MessageRef identifies one message inside one peer.
type MessageRef struct {
	Peer      Ref
	MessageID int
}

func (r MessageRef) String() string {
	return FormatMessageRef(r.Peer.String(), r.MessageID)
}

// FormatMessageRef returns the CLI message ref form: <peer-ref>:<message-id>.
func FormatMessageRef(peer string, messageID int) string {
	return fmt.Sprintf("%s:%d", peer, messageID)
}

// ParseMessageRef parses <peer-ref>:<message-id>. The peer side may itself
// contain colons, so the message id is split from the right.
func ParseMessageRef(s string) (MessageRef, error) {
	i := strings.LastIndex(s, ":")
	if i <= 0 || i == len(s)-1 {
		return MessageRef{}, errors.New("message ref must be <peer-ref>:<message-id>")
	}
	msgID, err := strconv.Atoi(s[i+1:])
	if err != nil || msgID <= 0 {
		return MessageRef{}, errors.New("message ref id must be positive")
	}
	peer, err := ParseRef(s[:i])
	if err != nil {
		return MessageRef{}, err
	}
	return MessageRef{Peer: peer, MessageID: msgID}, nil
}
