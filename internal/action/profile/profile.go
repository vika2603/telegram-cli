// Package profile contains profile command actions.
package profile

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// SetNameRequest is the normalized request for `tg profile set-name`.
type SetNameRequest struct {
	First   string
	Last    string
	LastSet bool
}

// SetNameFunc updates first and last name after request normalization.
type SetNameFunc func(context.Context, SetNameRequest) (output.ProfileRow, error)

// SetName validates and updates profile name.
func SetName(ctx context.Context, req SetNameRequest, update SetNameFunc) (output.ProfileRow, error) {
	if update == nil {
		return output.ProfileRow{}, fmt.Errorf("%w: profile set-name called without update function", command.ErrPrecondition)
	}
	return update(ctx, req)
}

// SetBioRequest is the raw request for `tg profile set-bio`.
type SetBioRequest struct {
	Bio   string
	Stdin io.Reader
}

// SetBioFunc updates profile bio after stdin normalization.
type SetBioFunc func(context.Context, string) (output.ProfileRow, error)

// SetBio validates and updates profile bio.
func SetBio(ctx context.Context, req SetBioRequest, update SetBioFunc) (output.ProfileRow, error) {
	if update == nil {
		return output.ProfileRow{}, fmt.Errorf("%w: profile set-bio called without update function", command.ErrPrecondition)
	}
	bio := req.Bio
	if bio == "-" {
		b, err := io.ReadAll(req.Stdin)
		if err != nil {
			return output.ProfileRow{}, err
		}
		bio = strings.TrimRight(string(b), "\n")
	}
	return update(ctx, bio)
}

// SetUsernameFunc updates the public username.
type SetUsernameFunc func(context.Context, string) (output.ProfileRow, error)

// SetUsername validates and updates the public username.
func SetUsername(ctx context.Context, username string, update SetUsernameFunc) (output.ProfileRow, error) {
	if update == nil {
		return output.ProfileRow{}, fmt.Errorf("%w: profile set-username called without update function", command.ErrPrecondition)
	}
	return update(ctx, username)
}

// SetStatusFunc updates online/offline visibility.
type SetStatusFunc func(context.Context, bool) (output.ProfileRow, error)

// SetStatus validates and updates online/offline visibility.
func SetStatus(ctx context.Context, state string, update SetStatusFunc) (output.ProfileRow, error) {
	if update == nil {
		return output.ProfileRow{}, fmt.Errorf("%w: profile set-status called without update function", command.ErrPrecondition)
	}
	if state != "online" && state != "offline" {
		return output.ProfileRow{}, fmt.Errorf("%w: state must be 'online' or 'offline'", command.ErrUsage)
	}
	return update(ctx, state == "offline")
}

// SetPhotoRequest is the raw request for `tg profile set-photo`.
type SetPhotoRequest struct {
	Path  string
	Stdin io.Reader
}

// SetPhotoFunc uploads and applies a new profile photo.
type SetPhotoFunc func(context.Context, string, io.Reader) (output.ProfileRow, error)

// SetPhoto validates and updates the profile photo.
func SetPhoto(ctx context.Context, req SetPhotoRequest, upload SetPhotoFunc) (output.ProfileRow, error) {
	if upload == nil {
		return output.ProfileRow{}, fmt.Errorf("%w: profile set-photo called without upload function", command.ErrPrecondition)
	}
	if req.Path != "-" {
		if _, err := os.Stat(req.Path); err != nil {
			return output.ProfileRow{}, fmt.Errorf("%w: cannot open photo: %s", command.ErrUsage, err.Error())
		}
	}
	return upload(ctx, req.Path, req.Stdin)
}

// DeletePhotoRequest is the raw request for `tg profile delete-photo`.
type DeletePhotoRequest struct {
	Yes      bool
	Prompter ui.Prompter
}

// DeletePhotoFunc removes the current profile photo.
type DeletePhotoFunc func(context.Context) error

// DeletePhoto confirms and deletes the current profile photo.
func DeletePhoto(ctx context.Context, req DeletePhotoRequest, delete DeletePhotoFunc) error {
	if delete == nil {
		return fmt.Errorf("%w: profile delete-photo called without delete function", command.ErrPrecondition)
	}
	if err := ui.ConfirmDestructive(req.Prompter, "delete current profile photo?", req.Yes); err != nil {
		return err
	}
	return delete(ctx)
}
