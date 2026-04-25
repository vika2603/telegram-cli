package output

// PeerObject is the canonical nested JSON shape for any Telegram peer.
type PeerObject struct {
	Ref      string `json:"ref"`
	ID       int64  `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Title    string `json:"title,omitempty"`
	Username string `json:"username,omitempty"`
	Link     string `json:"link,omitempty"`
}

// MessageSummary is the compact nested JSON shape used for last-message and
// write-result payloads.
type MessageSummary struct {
	Ref   string       `json:"ref,omitempty"`
	ID    int          `json:"id,omitempty"`
	Date  string       `json:"date,omitempty"`
	From  *PeerObject  `json:"from,omitempty"`
	Text  string       `json:"text,omitempty"`
	Media *MediaObject `json:"media,omitempty"`
}

// MediaObject is the nested JSON shape for message media metadata.
type MediaObject struct {
	Type string `json:"type"`
}

func peerObject(ref string, id int64, kind, title, username string) PeerObject {
	return PeerObject{
		Ref:      ref,
		ID:       id,
		Type:     kind,
		Title:    title,
		Username: username,
		Link:     peerLink(username),
	}
}

func peerLink(username string) string {
	if username == "" {
		return ""
	}
	return "https://t.me/" + username
}
