// Package session is the sole constructor of gotd's telegram.Client. All other
// internal packages interact with Telegram via the Client interface defined
// here and receive live values only inside a Run callback.
package session

import (
	"context"

	"github.com/gotd/td/tg"
	"go.uber.org/zap"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// Client is the narrow surface used by message and chat commands.
// The Client is valid ONLY inside a Run callback.
type Client interface {
	Invoker() tg.Invoker
	Self() tg.User
	PeerStore() *account.PeerStore
	ResolvePeer(ctx context.Context, r ref.Ref) (tg.InputPeerClass, error)
	RefreshPeer(ctx context.Context, r ref.Ref) (tg.InputPeerClass, error)
}

// Options carries the knobs session.Run needs. runtime/defaults builds this from
// the merged config/env/flag values.
type Options struct {
	Logger      *zap.Logger
	APIID       int
	APIHash     string
	Device      DeviceOptions
	FloodMode   FloodMode // fail | wait
	FloodMaxSec int       // cap for wait mode; 0 = unlimited
}

// DeviceOptions controls the identity Telegram shows for this authorization
// under Settings -> Devices.
type DeviceOptions struct {
	Model          string
	SystemVersion  string
	AppVersion     string
	SystemLangCode string
	LangCode       string
}

// FloodMode selects the FLOOD_WAIT policy.
type FloodMode int

const (
	FloodFail FloodMode = iota // return ErrFloodWait immediately
	FloodWait                  // wait up to FloodMaxSec, then return ErrFloodWait
)
