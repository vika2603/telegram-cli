package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/telegram/query"
	"github.com/gotd/td/telegram/query/contacts/blocked"
	"github.com/gotd/td/tg"

	actioncontact "github.com/vika2603/telegram-cli/internal/action/contact"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// ListContacts loads contact rows through contacts.getContacts or getBlocked.
func ListContacts(ctx context.Context, api *tg.Client, pm *peers.Manager, q actioncontact.ListQuery) ([]output.ContactRow, error) {
	if q.Blocked {
		return listBlockedContacts(ctx, api)
	}
	raw, err := api.ContactsGetContacts(ctx, 0)
	if err != nil {
		return nil, err
	}
	cc, ok := raw.AsModified()
	if !ok {
		return nil, nil
	}
	if pm != nil {
		_ = pm.Apply(ctx, cc.Users, nil)
	}
	mutual := make(map[int64]bool, len(cc.Contacts))
	for _, c := range cc.Contacts {
		mutual[c.UserID] = c.Mutual
	}
	out := make([]output.ContactRow, 0, len(cc.Users))
	for _, uRaw := range cc.Users {
		u, ok := uRaw.(*tg.User)
		if !ok {
			continue
		}
		row := userToContactRow(u, mutual[u.ID])
		if row.Bot && !q.Bots {
			continue
		}
		if q.MutualOnly && !row.Mutual {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

// AddContact resolves a phone number and adds it to the account contact list.
func AddContact(ctx context.Context, api *tg.Client, pm *peers.Manager, q actioncontact.AddQuery) (output.ContactRow, error) {
	u, err := pm.ResolvePhone(ctx, q.Phone)
	if err != nil {
		return output.ContactRow{}, fmt.Errorf("%w: resolve phone: %s", peer.ErrNotFound, err.Error())
	}
	req := &tg.ContactsAddContactRequest{
		ID:                       u.InputUser(),
		FirstName:                q.First,
		LastName:                 q.Last,
		Phone:                    q.Phone,
		AddPhonePrivacyException: q.Mutual,
	}
	if _, err := api.ContactsAddContact(ctx, req); err != nil {
		return output.ContactRow{}, err
	}
	tgu := u.Raw()
	return output.ContactRow{
		ID:        tgu.ID,
		FirstName: q.First,
		LastName:  q.Last,
		Username:  tgu.Username,
		Phone:     q.Phone,
		Mutual:    q.Mutual,
		Bot:       tgu.Bot,
	}, nil
}

// BlockContact blocks one resolved peer.
func BlockContact(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actioncontact.PeerQuery) error {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return err
	}
	_, err = api.ContactsBlock(ctx, &tg.ContactsBlockRequest{ID: resolved.InputPeer})
	return err
}

// DeleteContact deletes one user or bot contact.
func DeleteContact(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actioncontact.PeerQuery) error {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return err
	}
	if resolved.Kind != "user" && resolved.Kind != "bot" {
		return fmt.Errorf("%w: contact delete requires a user reference", command.ErrUsage)
	}
	iu := &tg.InputUser{UserID: resolved.ID, AccessHash: resolved.AccessHash}
	_, err = api.ContactsDeleteContacts(ctx, []tg.InputUserClass{iu})
	return err
}

// UnblockContact unblocks one resolved peer.
func UnblockContact(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actioncontact.PeerQuery) error {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return err
	}
	_, err = api.ContactsUnblock(ctx, &tg.ContactsUnblockRequest{ID: resolved.InputPeer})
	return err
}

func listBlockedContacts(ctx context.Context, api *tg.Client) ([]output.ContactRow, error) {
	var out []output.ContactRow
	err := query.GetBlocked(api).ForEach(ctx, func(_ context.Context, e blocked.Elem) error {
		pu, ok := e.Contact.PeerID.(*tg.PeerUser)
		if !ok {
			return nil
		}
		row := output.ContactRow{ID: pu.UserID, Blocked: true}
		if u, ok := e.Entities.User(pu.UserID); ok && u != nil {
			fillContactRowFromUser(&row, u)
		}
		out = append(out, row)
		return nil
	})
	return out, err
}

func userToContactRow(u *tg.User, mutual bool) output.ContactRow {
	row := output.ContactRow{Mutual: mutual}
	fillContactRowFromUser(&row, u)
	return row
}

func fillContactRowFromUser(row *output.ContactRow, u *tg.User) {
	row.ID = u.ID
	row.FirstName = u.FirstName
	row.LastName = u.LastName
	row.Username = u.Username
	row.Phone = u.Phone
	row.Bot = u.Bot
}
