package account

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"time"
)

// OpenRecentStore opens the small JSON store used by shell completion.
func OpenRecentStore(name string) (*PeerStore, error) {
	return &PeerStore{name: name, path: RecentFile(name)}, nil
}

// OpenRecentStoreReadOnly opens the recent-completion store only when it
// already exists. Completion treats a missing file as empty suggestions.
func OpenRecentStoreReadOnly(name string) (*PeerStore, error) {
	path := RecentFile(name)
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return &PeerStore{name: name, path: path}, nil
}

// PeerStore keeps only lightweight recent records for shell completion.
type PeerStore struct {
	name string
	path string
}

type RecentPeer struct {
	Ref       string    `json:"ref"`
	ID        int64     `json:"id,omitempty"`
	Kind      string    `json:"kind,omitempty"`
	Title     string    `json:"title,omitempty"`
	Username  string    `json:"username,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RecentMessage struct {
	Ref       string    `json:"ref"`
	PeerRef   string    `json:"peer_ref,omitempty"`
	MessageID int       `json:"message_id,omitempty"`
	Date      string    `json:"date,omitempty"`
	Text      string    `json:"text,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type recentData struct {
	Peers    map[string]RecentPeer    `json:"peers,omitempty"`
	Messages map[string]RecentMessage `json:"messages,omitempty"`
}

func (s *PeerStore) RecordRecentPeer(p RecentPeer) error {
	if s == nil || s.path == "" || p.Ref == "" {
		return nil
	}
	data, err := s.read()
	if err != nil {
		return err
	}
	if data.Peers == nil {
		data.Peers = make(map[string]RecentPeer)
	}
	p.UpdatedAt = time.Now().UTC()
	data.Peers[p.Ref] = p
	return s.write(data)
}

func (s *PeerStore) ListRecentPeers(query string, limit int) ([]RecentPeer, error) {
	data, err := s.read()
	if err != nil {
		return nil, err
	}
	rows := make([]RecentPeer, 0, len(data.Peers))
	return listRecent(rows, data.Peers, query, limit, func(p RecentPeer) string {
		return strings.Join([]string{p.Ref, p.Title, p.Username, p.Kind}, " ")
	}), nil
}

func (s *PeerStore) RecordRecentMessage(m RecentMessage) error {
	if s == nil || s.path == "" || m.Ref == "" {
		return nil
	}
	data, err := s.read()
	if err != nil {
		return err
	}
	if data.Messages == nil {
		data.Messages = make(map[string]RecentMessage)
	}
	m.UpdatedAt = time.Now().UTC()
	data.Messages[m.Ref] = m
	return s.write(data)
}

func (s *PeerStore) ListRecentMessages(query string, limit int) ([]RecentMessage, error) {
	data, err := s.read()
	if err != nil {
		return nil, err
	}
	rows := make([]RecentMessage, 0, len(data.Messages))
	return listRecent(rows, data.Messages, query, limit, func(m RecentMessage) string {
		return strings.Join([]string{m.Ref, m.PeerRef, m.Text}, " ")
	}), nil
}

func (s *PeerStore) read() (recentData, error) {
	var data recentData
	if s == nil || s.path == "" {
		return data, nil
	}
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return data, nil
	}
	if err != nil {
		return data, err
	}
	if len(b) == 0 {
		return data, nil
	}
	return data, json.Unmarshal(b, &data)
}

func (s *PeerStore) write(data recentData) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return AtomicWrite(s.path, LockFile(s.name), b)
}

func listRecent[T interface{ GetUpdatedAt() time.Time }](
	rows []T,
	items map[string]T,
	query string,
	limit int,
	searchable func(T) string,
) []T {
	if limit <= 0 {
		limit = 20
	}
	q := strings.ToLower(query)
	for _, row := range items {
		if q == "" || strings.Contains(strings.ToLower(searchable(row)), q) {
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].GetUpdatedAt().After(rows[j].GetUpdatedAt())
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows
}

func (p RecentPeer) GetUpdatedAt() time.Time { return p.UpdatedAt }

func (m RecentMessage) GetUpdatedAt() time.Time { return m.UpdatedAt }
