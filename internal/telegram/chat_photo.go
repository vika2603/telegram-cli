package telegram

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// SetChatPhoto sets or clears a supergroup/channel photo via
// channels.editPhoto. For set, Path is uploaded ("-" reads stdin); for clear,
// InputChatPhotoEmpty is sent.
func SetChatPhoto(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.PhotoQuery, stdin io.Reader) (output.ChatPhotoRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.ChatPhotoRow{}, err
	}
	inCh, ok := inputChannelFromPeer(resolved.InputPeer)
	if !ok {
		return output.ChatPhotoRow{}, fmt.Errorf("%w: a photo can only be set on a supergroup or channel", command.ErrUnsupported)
	}

	var photo tg.InputChatPhotoClass = &tg.InputChatPhotoEmpty{}
	action := "clear"
	if !q.Clear {
		up := uploader.NewUploader(api)
		var file tg.InputFileClass
		if q.Path == "-" {
			if stdin == nil {
				stdin = os.Stdin
			}
			file, err = up.FromReader(ctx, "photo", stdin)
		} else {
			file, err = up.FromPath(ctx, q.Path)
		}
		if err != nil {
			return output.ChatPhotoRow{}, err
		}
		uploaded := &tg.InputChatUploadedPhoto{}
		uploaded.SetFile(file)
		photo = uploaded
		action = "set"
	}

	if _, err := api.ChannelsEditPhoto(ctx, &tg.ChannelsEditPhotoRequest{Channel: inCh, Photo: photo}); err != nil {
		return output.ChatPhotoRow{}, err
	}
	return output.ChatPhotoRow{Action: action, Peer: output.PeerRefFromResolved(resolved)}, nil
}
