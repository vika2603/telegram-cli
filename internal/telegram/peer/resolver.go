// Package peer resolves parsed peer references into concrete Telegram peers,
// consulting the local bbolt cache first and falling back to gotd.
package peer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gotd/td/constant"
	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	gotdtgerr "github.com/gotd/td/tgerr"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// Resolved is the denormalized result. InputPeer is what gotd call sites
// need; the scalar fields are what output rows render.
type Resolved struct {
	InputPeer  tg.InputPeerClass
	Kind       string // "user" | "chat" | "channel" | "bot"
	ID         int64
	AccessHash int64
	Title      string
	Username   string
	Phone      string
}

// Resolver owns a peers.Manager and a PeerStore and converts ref.Ref
// values to Resolved. It is safe to reuse across commands within one
// session.Run callback, but MUST NOT outlive it.
type Resolver struct {
	mgr    *peers.Manager
	store  *account.PeerStore
	selfID int64
}

// New builds a Resolver. mgr must be non-nil; store may be nil (cache
// disabled).
func New(mgr *peers.Manager, store *account.PeerStore, selfID int64) (*Resolver, error) {
	if mgr == nil {
		return nil, errors.New("telegram peer: manager is nil")
	}
	return &Resolver{mgr: mgr, store: store, selfID: selfID}, nil
}

func (r *Resolver) Store() *account.PeerStore {
	if r == nil {
		return nil
	}
	return r.store
}

// Resolve converts ref to a Resolved. Cache-miss resolution calls the
// server. Returns ErrNotFound if the server reports no such
// peer; ErrAmbiguous if the ref is structurally ambiguous.
func (r *Resolver) Resolve(ctx context.Context, target ref.Ref) (Resolved, error) {
	var (
		out Resolved
		err error
	)
	switch target.Kind {
	case ref.RefKindMe, ref.RefKindSaved:
		out, err = r.resolveSelf(ctx)
	case ref.RefKindUsername:
		out, err = r.resolveUsername(ctx, target.Value)
	case ref.RefKindID:
		out, err = r.resolveID(ctx, target.ID)
	case ref.RefKindPeer:
		out, err = resolveDirectPeer(target)
	case ref.RefKindPhone:
		out, err = r.resolvePhone(ctx, target.Value)
	case ref.RefKindTMeLink:
		out, err = r.resolveTMe(ctx, target.Value)
	case ref.RefKindTGDeeplink:
		out, err = r.resolveDeeplink(ctx, target.Value)
	default:
		return Resolved{}, fmt.Errorf("%w: unsupported ref kind %v", command.ErrUsage, target.Kind)
	}
	if err != nil {
		return Resolved{}, err
	}
	_ = r.recordRecent(out)
	return out, nil
}

func resolveDirectPeer(target ref.Ref) (Resolved, error) {
	switch target.Value {
	case "user":
		return Resolved{
			InputPeer:  &tg.InputPeerUser{UserID: target.ID, AccessHash: target.AccessHash},
			Kind:       "user",
			ID:         target.ID,
			AccessHash: target.AccessHash,
		}, nil
	case "chat":
		return Resolved{
			InputPeer: &tg.InputPeerChat{ChatID: target.ID},
			Kind:      "chat",
			ID:        -target.ID,
		}, nil
	case "channel":
		return Resolved{
			InputPeer:  &tg.InputPeerChannel{ChannelID: target.ID, AccessHash: target.AccessHash},
			Kind:       "channel",
			ID:         -1_000_000_000_000 - target.ID,
			AccessHash: target.AccessHash,
		}, nil
	default:
		return Resolved{}, fmt.Errorf("%w: unsupported peer ref kind %q", command.ErrUsage, target.Value)
	}
}

func (r *Resolver) resolveSelf(_ context.Context) (Resolved, error) {
	return Resolved{
		InputPeer: &tg.InputPeerSelf{},
		Kind:      "user",
		ID:        r.selfID,
	}, nil
}

func (r *Resolver) recordRecent(p Resolved) error {
	if r == nil || r.store == nil {
		return nil
	}
	return r.store.RecordRecentPeer(account.RecentPeer{
		Ref:      preferredRef(p),
		ID:       p.ID,
		Kind:     p.Kind,
		Title:    p.Title,
		Username: p.Username,
	})
}

func preferredRef(r Resolved) string {
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

func (r *Resolver) resolveUsername(ctx context.Context, name string) (Resolved, error) {
	p, err := r.mgr.ResolveDomain(ctx, name)
	if err != nil {
		return Resolved{}, mapResolveErr(err, name)
	}
	return fromPeersPeer(p), nil
}

func (r *Resolver) resolveID(ctx context.Context, id int64) (Resolved, error) {
	if r.store == nil {
		return Resolved{}, fmt.Errorf("%w: id %d (no cache)", ErrCacheMiss, id)
	}
	hit, ok, err := lookupCachedByID(r.store, id)
	if err != nil {
		return Resolved{}, err
	}
	if ok {
		return hit, nil
	}
	p, err := r.mgr.ResolveTDLibID(ctx, constant.TDLibPeerID(id))
	if err != nil {
		return Resolved{}, mapResolveErr(err, fmt.Sprintf("%d", id))
	}
	return fromPeersPeer(p), nil
}

func (r *Resolver) resolvePhone(ctx context.Context, phone string) (Resolved, error) {
	p, err := r.mgr.ResolvePhone(ctx, phone)
	if err != nil {
		return Resolved{}, mapResolveErr(err, "+"+phone)
	}
	var peer peers.Peer = p
	return fromPeersPeer(peer), nil
}

func (r *Resolver) resolveTMe(ctx context.Context, path string) (Resolved, error) {
	// t.me/<username> resolves as a username. t.me/joinchat/X and t.me/+X
	// are invite links handled by join flows. t.me/c/<id>/<msg> is a
	// channel message ref and is not a peer target on its own.
	if path == "" {
		return Resolved{}, fmt.Errorf("%w: empty t.me path", command.ErrUsage)
	}
	if strings.HasPrefix(path, "joinchat/") || strings.HasPrefix(path, "+") {
		return Resolved{}, fmt.Errorf("%w: invite links are only valid for commands that join chats", command.ErrUnsupported)
	}
	if strings.HasPrefix(path, "c/") {
		return Resolved{}, fmt.Errorf("%w: t.me/c/ message links are not peer refs", command.ErrUsage)
	}
	// Take the leading path segment as the username.
	name := path
	if i := strings.IndexByte(path, '/'); i >= 0 {
		name = path[:i]
	}
	return r.resolveUsername(ctx, name)
}

func (r *Resolver) resolveDeeplink(ctx context.Context, body string) (Resolved, error) {
	// tg://resolve?domain=NAME maps to a username. Other deeplink shapes are
	// not peer refs.
	if !strings.HasPrefix(body, "resolve?") {
		return Resolved{}, fmt.Errorf("%w: only tg://resolve deeplinks are valid peer refs", command.ErrUnsupported)
	}
	q := body[len("resolve?"):]
	for _, kv := range strings.Split(q, "&") {
		if name, ok := strings.CutPrefix(kv, "domain="); ok {
			return r.resolveUsername(ctx, name)
		}
	}
	return Resolved{}, fmt.Errorf("%w: tg://resolve missing domain", command.ErrUsage)
}

// normalizeInputPeerID maps a gotd InputPeer to the signed-int64 ID
// convention `tg chat list` emits: users positive, legacy chats
// -ChatID, channels -1e12 - ChannelID. InputPeerSelf and any unknown
// variant return 0; callers fall back to peers.Peer.ID() so
// resolveSelf (which has no InputPeerUser yet) still reports a
// non-zero ID.
func normalizeInputPeerID(p tg.InputPeerClass) int64 {
	switch v := p.(type) {
	case *tg.InputPeerUser:
		return v.UserID
	case *tg.InputPeerChat:
		return -v.ChatID
	case *tg.InputPeerChannel:
		return -1_000_000_000_000 - v.ChannelID
	}
	return 0
}

// peerNotFoundRPCTypes are Telegram RPC error types that structurally
// mean "that ref does not resolve" and should surface as ErrPeerNotFound
// (exit 66) rather than a generic RPC error (exit 1).
var peerNotFoundRPCTypes = []string{
	"USERNAME_NOT_OCCUPIED",
	"USERNAME_INVALID",
	"CHANNEL_PRIVATE",
	"PEER_ID_INVALID",
	"CHAT_ID_INVALID",
	"USER_ID_INVALID",
}

func mapResolveErr(err error, ref string) error {
	var pnf *peers.PeerNotFoundError
	if errors.As(err, &pnf) {
		return fmt.Errorf("%w: %s", ErrNotFound, ref)
	}
	var phnf *peers.PhoneNotFoundError
	if errors.As(err, &phnf) {
		return fmt.Errorf("%w: %s", ErrNotFound, ref)
	}
	if gotdtgerr.Is(err, peerNotFoundRPCTypes...) {
		return fmt.Errorf("%w: %s", ErrNotFound, ref)
	}
	return err
}

func fromPeersPeer(p peers.Peer) Resolved {
	out := Resolved{
		InputPeer: p.InputPeer(),
	}
	out.ID = normalizeInputPeerID(out.InputPeer)
	if out.ID == 0 {
		out.ID = p.ID()
	}
	switch inp := out.InputPeer.(type) {
	case *tg.InputPeerUser:
		out.AccessHash = inp.AccessHash
	case *tg.InputPeerChannel:
		out.AccessHash = inp.AccessHash
	}
	switch v := p.(type) {
	case peers.User:
		out.Kind = "user"
		if _, ok := v.ToBot(); ok {
			out.Kind = "bot"
		}
		if n, ok := v.Username(); ok {
			out.Username = n
		}
		if ph, ok := v.Phone(); ok {
			out.Phone = ph
		}
		out.Title = userDisplayName(v)
	case peers.Chat:
		out.Kind = "chat"
		out.Title = v.VisibleName()
	case peers.Channel:
		if v.IsBroadcast() {
			out.Kind = "channel"
		} else {
			out.Kind = "chat"
		}
		if n, ok := v.Username(); ok {
			out.Username = n
		}
		out.Title = v.VisibleName()
	}
	return out
}

func userDisplayName(u peers.User) string {
	first, _ := u.FirstName()
	last, _ := u.LastName()
	switch {
	case first != "" && last != "":
		return first + " " + last
	case first != "":
		return first
	case last != "":
		return last
	}
	if n, ok := u.Username(); ok {
		return "@" + n
	}
	return fmt.Sprintf("user#%d", u.ID())
}

// lookupCachedByID queries the peer cache by numeric ID.
// ID-only lookup returning raw bytes is a forward-compat hook; the current
// store does not yet deserialize cached peers back into Resolved.
func lookupCachedByID(store *account.PeerStore, id int64) (Resolved, bool, error) {
	switch {
	case id > 0:
		u, ok, err := store.FindUser(context.Background(), id)
		if err != nil || !ok {
			return Resolved{}, ok, err
		}
		kind := "user"
		if u.Bot {
			kind = "bot"
		}
		return Resolved{
			InputPeer:  &tg.InputPeerUser{UserID: u.ID, AccessHash: u.AccessHash},
			Kind:       kind,
			ID:         u.ID,
			AccessHash: u.AccessHash,
			Title:      userTitleFromTG(u),
			Username:   u.Username,
			Phone:      u.Phone,
		}, true, nil
	case id < -1_000_000_000_000:
		channelID := -1_000_000_000_000 - id
		c, ok, err := store.FindChannel(context.Background(), channelID)
		if err != nil || !ok {
			return Resolved{}, ok, err
		}
		kind := "chat"
		if c.Broadcast {
			kind = "channel"
		}
		return Resolved{
			InputPeer:  &tg.InputPeerChannel{ChannelID: c.ID, AccessHash: c.AccessHash},
			Kind:       kind,
			ID:         id,
			AccessHash: c.AccessHash,
			Title:      c.Title,
			Username:   c.Username,
		}, true, nil
	case id < 0:
		chatID := -id
		c, ok, err := store.FindChat(context.Background(), chatID)
		if err != nil || !ok {
			return Resolved{}, ok, err
		}
		return Resolved{
			InputPeer: &tg.InputPeerChat{ChatID: c.ID},
			Kind:      "chat",
			ID:        id,
			Title:     c.Title,
		}, true, nil
	default:
		return Resolved{}, false, nil
	}
}

func userTitleFromTG(u *tg.User) string {
	switch {
	case u.FirstName != "" && u.LastName != "":
		return u.FirstName + " " + u.LastName
	case u.FirstName != "":
		return u.FirstName
	case u.LastName != "":
		return u.LastName
	case u.Username != "":
		return "@" + u.Username
	default:
		return fmt.Sprintf("user#%d", u.ID)
	}
}
