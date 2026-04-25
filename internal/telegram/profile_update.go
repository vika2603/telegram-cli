package telegram

import (
	"context"

	"github.com/gotd/td/tg"

	actionprofile "github.com/vika2603/telegram-cli/internal/action/profile"
	"github.com/vika2603/telegram-cli/internal/output"
)

// UpdateProfileName updates first and optional last name.
func UpdateProfileName(ctx context.Context, api *tg.Client, args actionprofile.SetNameRequest) (output.ProfileRow, error) {
	req := &tg.AccountUpdateProfileRequest{}
	req.SetFirstName(args.First)
	if args.LastSet {
		req.SetLastName(args.Last)
	}
	u, err := api.AccountUpdateProfile(ctx, req)
	if err != nil {
		return output.ProfileRow{}, err
	}
	first, last := firstLast(u)
	return output.ProfileRow{Action: "set-name", FirstName: first, LastName: last}, nil
}

// UpdateProfileBio updates the account bio.
func UpdateProfileBio(ctx context.Context, api *tg.Client, bio string) (output.ProfileRow, error) {
	req := &tg.AccountUpdateProfileRequest{}
	req.SetAbout(bio)
	if _, err := api.AccountUpdateProfile(ctx, req); err != nil {
		return output.ProfileRow{}, err
	}
	return output.ProfileRow{Action: "set-bio", Bio: bio}, nil
}

// UpdateProfileUsername updates or clears the public username.
func UpdateProfileUsername(ctx context.Context, api *tg.Client, username string) (output.ProfileRow, error) {
	uc, err := api.AccountUpdateUsername(ctx, username)
	if err != nil {
		return output.ProfileRow{}, err
	}
	got := ""
	if user, ok := uc.(*tg.User); ok {
		got = user.Username
	}
	return output.ProfileRow{Action: "set-username", Username: got}, nil
}

// UpdateProfileStatus updates online/offline visibility.
func UpdateProfileStatus(ctx context.Context, api *tg.Client, offline bool) (output.ProfileRow, error) {
	if _, err := api.AccountUpdateStatus(ctx, offline); err != nil {
		return output.ProfileRow{}, err
	}
	state := "online"
	if offline {
		state = "offline"
	}
	return output.ProfileRow{Action: "set-status", Status: state}, nil
}

func firstLast(u tg.UserClass) (string, string) {
	if user, ok := u.(*tg.User); ok {
		return user.FirstName, user.LastName
	}
	return "", ""
}
