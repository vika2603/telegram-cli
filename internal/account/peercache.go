package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"go.etcd.io/bbolt"
	bbolterrors "go.etcd.io/bbolt/errors"
)

// Bucket names used inside peers.db.
const (
	BucketPeers              = "peers"
	BucketPeerStorage        = "peer_storage"
	BucketPeerPhones         = "peer_phones"
	BucketPeerContacts       = "peer_contacts"
	BucketPeerCacheUsers     = "peer_cache_users"
	BucketPeerCacheUserFulls = "peer_cache_user_fulls"
	BucketPeerCacheChats     = "peer_cache_chats"
	BucketPeerCacheChatFulls = "peer_cache_chat_fulls"
	BucketPeerCacheChannels  = "peer_cache_channels"
	BucketPeerCacheChanFulls = "peer_cache_channel_fulls"
	BucketRecentPeers        = "recent_peers"
	BucketRecentMessages     = "recent_messages"
	BucketSelf               = "self"
)

// BoltOpenTimeout is the lock acquisition timeout for bbolt opens. Defined as
// a var so tests can compress it; production code must not mutate it.
var BoltOpenTimeout = 5 * time.Second

// OpenPeersDB opens peers.db; lock failure maps to ErrBusy.
func OpenPeersDB(name string) (*bbolt.DB, error) {
	return openBolt(PeersDB(name), BucketPeers)
}

// OpenPeersDBReadOnly opens peers.db without creating missing buckets. It is
// intended for completion, where cache misses should be silent and cheap.
func OpenPeersDBReadOnly(name string) (*bbolt.DB, error) {
	db, err := bbolt.Open(PeersDB(name), 0600, &bbolt.Options{
		ReadOnly: true,
		Timeout:  BoltOpenTimeout,
	})
	if errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if errors.Is(err, bbolterrors.ErrTimeout) {
		return nil, fmt.Errorf("open %s: %w", PeersDB(name), ErrBusy)
	}
	return db, err
}

func openBolt(path string, initialBucket string) (*bbolt.DB, error) {
	db, err := bbolt.Open(path, 0600, &bbolt.Options{
		Timeout: BoltOpenTimeout,
	})
	if errors.Is(err, bbolterrors.ErrTimeout) {
		return nil, fmt.Errorf("open %s: %w", path, ErrBusy)
	}
	if err != nil {
		return nil, err
	}
	if err := ensureBuckets(db, initialBucket); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func ensureBuckets(db *bbolt.DB, names ...string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		for _, n := range names {
			if _, err := tx.CreateBucketIfNotExists([]byte(n)); err != nil {
				return err
			}
		}
		return nil
	})
}

// PeerStore wraps peers.db. The core exposes the constructor, DB accessor, and
// CacheSelf. Richer peer-resolution methods are attached elsewhere.
type PeerStore struct {
	db *bbolt.DB
}

func NewPeerStore(db *bbolt.DB) *PeerStore { return &PeerStore{db: db} }

func (s *PeerStore) DB() *bbolt.DB { return s.db }

type peerStoragePhoneRecord struct {
	Key   peers.Key   `json:"key"`
	Value peers.Value `json:"value"`
}

func (s *PeerStore) Save(_ context.Context, key peers.Key, value peers.Value) error {
	if s == nil || s.db == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(BucketPeerStorage))
		if err != nil {
			return err
		}
		return b.Put([]byte(peerStorageKey(key)), data)
	})
}

func (s *PeerStore) Find(_ context.Context, key peers.Key) (peers.Value, bool, error) {
	if s == nil || s.db == nil {
		return peers.Value{}, false, nil
	}
	var out peers.Value
	found := false
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketPeerStorage))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(peerStorageKey(key)))
		if v == nil {
			return nil
		}
		found = true
		return json.Unmarshal(v, &out)
	})
	return out, found, err
}

func (s *PeerStore) SavePhone(ctx context.Context, phone string, key peers.Key) error {
	if s == nil || s.db == nil {
		return nil
	}
	value, found, err := s.Find(ctx, key)
	if err != nil {
		return err
	}
	if !found {
		value = peers.Value{}
	}
	data, err := json.Marshal(peerStoragePhoneRecord{Key: key, Value: value})
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(BucketPeerPhones))
		if err != nil {
			return err
		}
		return b.Put([]byte(phone), data)
	})
}

func (s *PeerStore) FindPhone(_ context.Context, phone string) (peers.Key, peers.Value, bool, error) {
	if s == nil || s.db == nil {
		return peers.Key{}, peers.Value{}, false, nil
	}
	var out peerStoragePhoneRecord
	found := false
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketPeerPhones))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(phone))
		if v == nil {
			return nil
		}
		found = true
		return json.Unmarshal(v, &out)
	})
	return out.Key, out.Value, found, err
}

func (s *PeerStore) GetContactsHash(_ context.Context) (int64, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	var out int64
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketPeerContacts))
		if b == nil {
			return nil
		}
		v := b.Get([]byte("hash"))
		if v == nil {
			return nil
		}
		out, _ = strconv.ParseInt(string(v), 10, 64)
		return nil
	})
	return out, err
}

func (s *PeerStore) SaveContactsHash(_ context.Context, hash int64) error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(BucketPeerContacts))
		if err != nil {
			return err
		}
		return b.Put([]byte("hash"), []byte(strconv.FormatInt(hash, 10)))
	})
}

func peerStorageKey(key peers.Key) string {
	return key.Prefix + strconv.FormatInt(key.ID, 10)
}

func (s *PeerStore) SaveUsers(_ context.Context, users ...*tg.User) error {
	return saveTGObjects(s, BucketPeerCacheUsers, users, func(u *tg.User) int64 { return u.ID })
}

func (s *PeerStore) SaveUserFulls(_ context.Context, users ...*tg.UserFull) error {
	return saveTGObjects(s, BucketPeerCacheUserFulls, users, func(u *tg.UserFull) int64 { return u.ID })
}

func (s *PeerStore) FindUser(_ context.Context, id int64) (*tg.User, bool, error) {
	return findTGObject(s, BucketPeerCacheUsers, id, func() *tg.User { return new(tg.User) })
}

func (s *PeerStore) FindUserFull(_ context.Context, id int64) (*tg.UserFull, bool, error) {
	return findTGObject(s, BucketPeerCacheUserFulls, id, func() *tg.UserFull { return new(tg.UserFull) })
}

func (s *PeerStore) SaveChats(_ context.Context, chats ...*tg.Chat) error {
	return saveTGObjects(s, BucketPeerCacheChats, chats, func(c *tg.Chat) int64 { return c.ID })
}

func (s *PeerStore) SaveChatFulls(_ context.Context, chats ...*tg.ChatFull) error {
	return saveTGObjects(s, BucketPeerCacheChatFulls, chats, func(c *tg.ChatFull) int64 { return c.ID })
}

func (s *PeerStore) FindChat(_ context.Context, id int64) (*tg.Chat, bool, error) {
	return findTGObject(s, BucketPeerCacheChats, id, func() *tg.Chat { return new(tg.Chat) })
}

func (s *PeerStore) FindChatFull(_ context.Context, id int64) (*tg.ChatFull, bool, error) {
	return findTGObject(s, BucketPeerCacheChatFulls, id, func() *tg.ChatFull { return new(tg.ChatFull) })
}

func (s *PeerStore) SaveChannels(_ context.Context, channels ...*tg.Channel) error {
	return saveTGObjects(s, BucketPeerCacheChannels, channels, func(c *tg.Channel) int64 { return c.ID })
}

func (s *PeerStore) SaveChannelFulls(_ context.Context, channels ...*tg.ChannelFull) error {
	return saveTGObjects(s, BucketPeerCacheChanFulls, channels, func(c *tg.ChannelFull) int64 { return c.ID })
}

func (s *PeerStore) FindChannel(_ context.Context, id int64) (*tg.Channel, bool, error) {
	return findTGObject(s, BucketPeerCacheChannels, id, func() *tg.Channel { return new(tg.Channel) })
}

func (s *PeerStore) FindChannelFull(_ context.Context, id int64) (*tg.ChannelFull, bool, error) {
	return findTGObject(s, BucketPeerCacheChanFulls, id, func() *tg.ChannelFull { return new(tg.ChannelFull) })
}

type peerCacheObject interface {
	*tg.User | *tg.UserFull | *tg.Chat | *tg.ChatFull | *tg.Channel | *tg.ChannelFull
	Encode(*bin.Buffer) error
}

type peerCacheDecodeObject interface {
	peerCacheObject
	Decode(*bin.Buffer) error
}

func saveTGObjects[T peerCacheObject](s *PeerStore, bucket string, items []T, idOf func(T) int64) error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		for _, item := range items {
			if item == nil {
				continue
			}
			var buf bin.Buffer
			if err := item.Encode(&buf); err != nil {
				return err
			}
			data := buf.Copy()
			if err := b.Put([]byte(strconv.FormatInt(idOf(item), 10)), data); err != nil {
				return err
			}
		}
		return nil
	})
}

func findTGObject[T peerCacheDecodeObject](s *PeerStore, bucket string, id int64, newT func() T) (T, bool, error) {
	if s == nil || s.db == nil {
		var zero T
		return zero, false, nil
	}
	out := newT()
	found := false
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(strconv.FormatInt(id, 10)))
		if v == nil {
			return nil
		}
		found = true
		buf := bin.Buffer{Buf: append([]byte(nil), v...)}
		if err := out.Decode(&buf); err != nil {
			found = false
		}
		return nil
	})
	if err != nil {
		var zero T
		return zero, false, err
	}
	if !found {
		var zero T
		return zero, false, nil
	}
	return out, true, nil
}

// CacheSelf persists the authenticated user under BucketSelf. JSON encoded
// so the bucket is inspectable without a custom decoder.
func (s *PeerStore) CacheSelf(u *tg.User) error {
	if u == nil {
		return errors.New("CacheSelf: user is nil")
	}
	data, err := json.Marshal(u)
	if err != nil {
		return fmt.Errorf("marshal self: %w", err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(BucketSelf))
		if err != nil {
			return fmt.Errorf("bucket %s: %w", BucketSelf, err)
		}
		return b.Put([]byte("id"), data)
	})
}

// LookupByID returns the cached peer row as a raw byte slice keyed by
// the decimal string of id, or (nil, false, nil) on miss.
func (s *PeerStore) LookupByID(id int64) ([]byte, bool, error) {
	if s == nil || s.db == nil {
		return nil, false, nil
	}
	var out []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketPeers))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(strconv.FormatInt(id, 10)))
		if v == nil {
			return nil
		}
		out = append([]byte(nil), v...)
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return out, out != nil, nil
}

// RecentPeer is a lightweight completion record, independent from gotd's
// serialized cache schema.
type RecentPeer struct {
	Ref       string    `json:"ref"`
	ID        int64     `json:"id,omitempty"`
	Kind      string    `json:"kind,omitempty"`
	Title     string    `json:"title,omitempty"`
	Username  string    `json:"username,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RecentMessage is a lightweight message-ref completion record.
type RecentMessage struct {
	Ref       string    `json:"ref"`
	PeerRef   string    `json:"peer_ref,omitempty"`
	MessageID int       `json:"message_id,omitempty"`
	Date      string    `json:"date,omitempty"`
	Text      string    `json:"text,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *PeerStore) RecordRecentPeer(p RecentPeer) error {
	if s == nil || s.db == nil || p.Ref == "" {
		return nil
	}
	p.UpdatedAt = time.Now().UTC()
	return putJSON(s.db, BucketRecentPeers, p.Ref, p)
}

func (s *PeerStore) ListRecentPeers(query string, limit int) ([]RecentPeer, error) {
	var rows []RecentPeer
	err := listJSON(s.db, BucketRecentPeers, query, limit, func(p RecentPeer) (RecentPeer, string) {
		return p, strings.Join([]string{p.Ref, p.Title, p.Username, p.Kind}, " ")
	}, &rows)
	return rows, err
}

func (s *PeerStore) RecordRecentMessage(m RecentMessage) error {
	if s == nil || s.db == nil || m.Ref == "" {
		return nil
	}
	m.UpdatedAt = time.Now().UTC()
	return putJSON(s.db, BucketRecentMessages, m.Ref, m)
}

func (s *PeerStore) ListRecentMessages(query string, limit int) ([]RecentMessage, error) {
	var rows []RecentMessage
	err := listJSON(s.db, BucketRecentMessages, query, limit, func(m RecentMessage) (RecentMessage, string) {
		return m, strings.Join([]string{m.Ref, m.PeerRef, m.Text}, " ")
	}, &rows)
	return rows, err
}

func putJSON[T any](db *bbolt.DB, bucket, key string, value T) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		return b.Put([]byte(key), data)
	})
}

func listJSON[T interface{ GetUpdatedAt() time.Time }, R any](
	db *bbolt.DB,
	bucket string,
	query string,
	limit int,
	adapt func(R) (T, string),
	out *[]T,
) error {
	if db == nil {
		return nil
	}
	if limit <= 0 {
		limit = 20
	}
	q := strings.ToLower(query)
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, v []byte) error {
			var raw R
			if err := json.Unmarshal(v, &raw); err != nil {
				return err
			}
			row, searchable := adapt(raw)
			if q == "" || strings.Contains(strings.ToLower(searchable), q) {
				*out = append(*out, row)
			}
			return nil
		})
	})
	if err != nil {
		return err
	}
	sort.Slice(*out, func(i, j int) bool {
		return (*out)[i].GetUpdatedAt().After((*out)[j].GetUpdatedAt())
	})
	if len(*out) > limit {
		*out = (*out)[:limit]
	}
	return nil
}

func (p RecentPeer) GetUpdatedAt() time.Time { return p.UpdatedAt }

func (m RecentMessage) GetUpdatedAt() time.Time { return m.UpdatedAt }

var _ peers.Storage = (*PeerStore)(nil)
var _ peers.Cache = (*PeerStore)(nil)
