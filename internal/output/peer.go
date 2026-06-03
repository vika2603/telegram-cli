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
	Ref       string          `json:"ref,omitempty"`
	ID        int             `json:"id,omitempty"`
	Date      string          `json:"date,omitempty"`
	EditDate  string          `json:"edit_date,omitempty"`
	GroupedID int64           `json:"grouped_id,omitempty"`
	From      *PeerObject     `json:"from,omitempty"`
	Text      string          `json:"text,omitempty"`
	Media     *MediaObject    `json:"media,omitempty"`
	ReplyTo   *ReplyInfo      `json:"reply_to,omitempty"`
	Forward   *ForwardInfo    `json:"forward,omitempty"`
	Entities  []MessageEntity `json:"entities,omitempty"`
	Buttons   []MessageButton `json:"buttons,omitempty"`
	Reactions []ReactionCount `json:"reactions,omitempty"`
}

// MediaObject is the nested JSON shape for message media metadata. Type is
// always set; the rest are filled in by `msg info` per media kind (omitempty).
type MediaObject struct {
	Type        string    `json:"type"`
	FileName    string    `json:"file_name,omitempty"`    // documents
	Size        int64     `json:"size,omitempty"`         // bytes
	MIME        string    `json:"mime,omitempty"`         // documents
	Width       int       `json:"width,omitempty"`        // photo/video
	Height      int       `json:"height,omitempty"`       // photo/video
	Duration    int       `json:"duration,omitempty"`     // video/audio/voice, seconds
	Performer   string    `json:"performer,omitempty"`    // audio
	AudioTitle  string    `json:"audio_title,omitempty"`  // audio
	Emoji       string    `json:"emoji,omitempty"`        // sticker
	StickerType string    `json:"sticker_type,omitempty"` // sticker: static|animated|video
	Title       string    `json:"title,omitempty"`        // web_page
	URL         string    `json:"url,omitempty"`          // web_page
	SiteName    string    `json:"site_name,omitempty"`    // web_page
	Description string    `json:"description,omitempty"`  // web_page
	Poll        *PollInfo `json:"poll,omitempty"`
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
