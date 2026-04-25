package session

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	gofastererrors "github.com/go-faster/errors"
	"github.com/gotd/td/tg"
	gotdtgerr "github.com/gotd/td/tgerr"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/ref"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

func TestRun_rejectsEXPIREDBeforeDial(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	acct := &account.Account{
		Meta: account.Meta{Name: "alice", State: account.StateEXPIRED, APIID: 1, APIHash: "h"},
		Dir:  account.AccountDir("alice"),
		Lock: account.LockFile("alice"),
		Sess: account.SessionFile("alice"),
	}
	called := false
	err := Run(context.Background(), acct, Options{}, func(_ context.Context, _ Client) error {
		called = true
		return nil
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAuth)
	require.False(t, called)
}

type fakeSelfProbe struct {
	user   *tg.User
	err    error
	calls  int
	onCall func(context.Context)
}

func (f *fakeSelfProbe) Self(ctx context.Context) (*tg.User, error) {
	f.calls++
	if f.onCall != nil {
		f.onCall(ctx)
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.user, nil
}

type fakeClient struct{ self tg.User }

func (f *fakeClient) Invoker() tg.Invoker { return nil }
func (f *fakeClient) Self() tg.User       { return f.self }
func (f *fakeClient) PeerStore() *account.PeerStore {
	return nil
}
func (f *fakeClient) ResolvePeer(context.Context, ref.Ref) (tg.InputPeerClass, error) {
	return nil, errors.New("not used")
}
func (f *fakeClient) RefreshPeer(context.Context, ref.Ref) (tg.InputPeerClass, error) {
	return nil, errors.New("not used")
}

func TestRunLifecycle_callbackSeesLiveClientAndCleanShutdown(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, account.AddAccount(account.Meta{
		Name: "alice", State: account.StateNEW, APIID: 1, APIHash: "h",
	}))
	acct, err := account.LoadAccount("alice")
	require.NoError(t, err)
	probe := &fakeSelfProbe{user: &tg.User{ID: 42, Username: "alice"}}
	var fnCalls int
	var fnClient Client
	err = runLifecycle(context.Background(), acct, Options{}, probe,
		func(self tg.User) Client { return &fakeClient{self: self} },
		func(_ context.Context, cl Client) error {
			fnCalls++
			fnClient = cl
			return nil
		})
	require.NoError(t, err)
	require.Equal(t, 1, probe.calls)
	require.Equal(t, 1, fnCalls)
	require.NotNil(t, fnClient)
	require.Equal(t, int64(42), fnClient.Self().ID)
	require.Equal(t, account.StateNEW, acct.Meta.State)
}

func TestRunLifecycle_ctxCancelMidCallbackHaltsCleanly(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, account.AddAccount(account.Meta{
		Name: "alice", State: account.StateAUTHED, APIID: 1, APIHash: "h",
	}))
	acct, err := account.LoadAccount("alice")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	probe := &fakeSelfProbe{user: &tg.User{ID: 42}}

	fnEntered := make(chan struct{})
	fnReturned := make(chan error, 1)
	go func() {
		fnReturned <- runLifecycle(ctx, acct, Options{}, probe,
			func(self tg.User) Client { return &fakeClient{self: self} },
			func(ctx context.Context, _ Client) error {
				close(fnEntered)
				<-ctx.Done()
				return ctx.Err()
			})
	}()
	select {
	case <-fnEntered:
	case <-time.After(time.Second):
		t.Fatal("fn did not enter within 1s")
	}
	cancel()
	select {
	case err = <-fnReturned:
	case <-time.After(time.Second):
		t.Fatal("runLifecycle did not return within 1s of cancel")
	}
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, probe.calls)
	require.Equal(t, account.StateAUTHED, acct.Meta.State)
}

func TestRunLifecycle_unauthorizedFlipsToEXPIRED(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, account.AddAccount(account.Meta{
		Name: "alice", State: account.StateAUTHED, APIID: 1, APIHash: "h",
	}))
	acct, err := account.LoadAccount("alice")
	require.NoError(t, err)
	authErr := gotdtgerr.New(401, "UNAUTHORIZED")
	probe := &fakeSelfProbe{err: authErr}
	var fnCalled bool
	err = runLifecycle(context.Background(), acct, Options{}, probe,
		func(self tg.User) Client { return &fakeClient{self: self} },
		func(_ context.Context, _ Client) error { fnCalled = true; return nil })
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAuth)
	require.False(t, fnCalled)
	require.Equal(t, account.StateEXPIRED, acct.Meta.State)
	got, rerr := account.ReadMeta("alice")
	require.NoError(t, rerr)
	require.Equal(t, account.StateEXPIRED, got.State)
}

func TestRunLifecycle_fnReturnsAuthErrorFlipsToEXPIRED(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, account.AddAccount(account.Meta{
		Name: "alice", State: account.StateAUTHED, APIID: 1, APIHash: "h",
	}))
	acct, err := account.LoadAccount("alice")
	require.NoError(t, err)
	probe := &fakeSelfProbe{user: &tg.User{ID: 42}}
	authErr := gotdtgerr.New(401, "UNAUTHORIZED")
	err = runLifecycle(context.Background(), acct, Options{}, probe,
		func(self tg.User) Client { return &fakeClient{self: self} },
		func(_ context.Context, _ Client) error { return authErr })
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAuth)
	require.Equal(t, account.StateEXPIRED, acct.Meta.State)
	got, rerr := account.ReadMeta("alice")
	require.NoError(t, rerr)
	require.Equal(t, account.StateEXPIRED, got.State)
}

func TestStripCallbackPrefix_unwrapsGotdCallbackWrap(t *testing.T) {
	// gotd's telegram.Client.Run wraps every error escaping the user
	// callback with `go-faster/errors.Wrap(err, "callback")`, which
	// produces "callback: <inner>" at the user-visible layer. Our CLI
	// strips that single layer so error output is clean; sentinels stay
	// reachable via errors.Is.
	inner := fmt.Errorf("%w: @missing", peer.ErrNotFound)
	wrapped := gofastererrors.Wrap(inner, "callback")
	require.Contains(t, wrapped.Error(), "callback: ")

	stripped := stripCallbackPrefix(wrapped)
	require.NotContains(t, stripped.Error(), "callback: ")
	require.ErrorIs(t, stripped, peer.ErrNotFound)
}

func TestStripCallbackPrefix_leavesOtherErrorsAlone(t *testing.T) {
	plain := errors.New("plain error without callback wrap")
	require.Same(t, plain, stripCallbackPrefix(plain))
	require.NoError(t, stripCallbackPrefix(nil))
}

func TestRunLifecycle_nonAuthErrorDoesNotFlipState(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, account.AddAccount(account.Meta{
		Name: "alice", State: account.StateAUTHED, APIID: 1, APIHash: "h",
	}))
	acct, err := account.LoadAccount("alice")
	require.NoError(t, err)
	transient := errors.New("transient network error")
	probe := &fakeSelfProbe{err: transient}
	err = runLifecycle(context.Background(), acct, Options{}, probe,
		func(self tg.User) Client { return &fakeClient{self: self} },
		func(_ context.Context, _ Client) error { return nil })
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrAuth)
	require.ErrorIs(t, err, transient)
	require.Equal(t, account.StateAUTHED, acct.Meta.State)
	got, rerr := account.ReadMeta("alice")
	require.NoError(t, rerr)
	require.Equal(t, account.StateAUTHED, got.State)
}
