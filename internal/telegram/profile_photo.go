package telegram

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

// UploadProfilePhoto uploads and applies a new profile photo.
func UploadProfilePhoto(ctx context.Context, api *tg.Client, path string, stdin io.Reader) (output.ProfileRow, error) {
	up := uploader.NewUploader(api)
	var (
		file tg.InputFileClass
		err  error
	)
	if path == "-" {
		if stdin == nil {
			stdin = os.Stdin
		}
		file, err = up.FromReader(ctx, "photo", stdin)
	} else {
		file, err = up.FromPath(ctx, path)
	}
	if err != nil {
		return output.ProfileRow{}, err
	}
	req := &tg.PhotosUploadProfilePhotoRequest{}
	req.SetFile(file)
	p, err := api.PhotosUploadProfilePhoto(ctx, req)
	if err != nil {
		return output.ProfileRow{}, err
	}
	return output.ProfileRow{Action: "set-photo", PhotoID: extractPhotoID(p)}, nil
}

// DeleteProfilePhoto removes the current profile photo.
func DeleteProfilePhoto(ctx context.Context, api *tg.Client) error {
	self, err := api.UsersGetFullUser(ctx, &tg.InputUserSelf{})
	if err != nil {
		return err
	}
	cur := currentPhotoID(self)
	if cur == nil {
		return fmt.Errorf("%w: no profile photo to delete", command.ErrPrecondition)
	}
	_, err = api.PhotosDeletePhotos(ctx, []tg.InputPhotoClass{cur})
	return err
}

func extractPhotoID(p *tg.PhotosPhoto) int64 {
	if p == nil {
		return 0
	}
	if ph, ok := p.Photo.(*tg.Photo); ok {
		return ph.ID
	}
	return 0
}

func currentPhotoID(full *tg.UsersUserFull) tg.InputPhotoClass {
	if full == nil {
		return nil
	}
	p, ok := full.FullUser.GetProfilePhoto()
	if !ok {
		return nil
	}
	photo, ok := p.(*tg.Photo)
	if !ok {
		return nil
	}
	return &tg.InputPhoto{
		ID:            photo.ID,
		AccessHash:    photo.AccessHash,
		FileReference: photo.FileReference,
	}
}
