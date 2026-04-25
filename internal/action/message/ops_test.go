package message_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestSend_NormalizesStdinText(t *testing.T) {
	rows, err := actionmessage.Send(context.Background(), actionmessage.SendRequest{
		RawRef: "@alice",
		Text:   "-",
		Stdin:  strings.NewReader("hello\n"),
	}, func(_ context.Context, q actionmessage.SendQuery) ([]output.SendResultRow, error) {
		require.Equal(t, "alice", q.Ref.Value)
		require.Equal(t, "hello", q.Text)
		return []output.SendResultRow{{Action: "send", MessageID: 1}}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "send", rows[0].Action)
}

func TestSend_RejectsConflictingStdin(t *testing.T) {
	_, err := actionmessage.Send(context.Background(), actionmessage.SendRequest{
		RawRef: "@alice",
		Text:   "-",
		Files:  []string{"-"},
		Stdin:  strings.NewReader("body"),
	}, func(context.Context, actionmessage.SendQuery) ([]output.SendResultRow, error) {
		t.Fatal("send function must not run for invalid stdin flags")
		return nil, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestSend_FileUsesTextAsCaption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.pdf")
	require.NoError(t, os.WriteFile(path, []byte("pdf"), 0600))

	_, err := actionmessage.Send(context.Background(), actionmessage.SendRequest{
		RawRef: "@alice",
		Text:   "see attached",
		Files:  []string{path},
	}, func(_ context.Context, q actionmessage.SendQuery) ([]output.SendResultRow, error) {
		require.Equal(t, "see attached", q.Text)
		require.Equal(t, []actionmessage.Attachment{{Path: path, Name: "report.pdf"}}, q.Attachments)
		return []output.SendResultRow{{Action: "send", MessageID: 1}}, nil
	})
	require.NoError(t, err)
}

func TestSend_AllowsRepeatedFilesAndNames(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.bin")
	b := filepath.Join(dir, "b.bin")
	require.NoError(t, os.WriteFile(a, []byte("a"), 0600))
	require.NoError(t, os.WriteFile(b, []byte("b"), 0600))

	_, err := actionmessage.Send(context.Background(), actionmessage.SendRequest{
		RawRef: "@alice",
		Files:  []string{a, b},
		Names:  []string{"one.dat", "two.dat"},
	}, func(_ context.Context, q actionmessage.SendQuery) ([]output.SendResultRow, error) {
		require.Equal(t, []actionmessage.Attachment{
			{Path: a, Name: "one.dat"},
			{Path: b, Name: "two.dat"},
		}, q.Attachments)
		return []output.SendResultRow{{Action: "send", MessageID: 1}}, nil
	})
	require.NoError(t, err)
}

func TestSend_RejectsMismatchedNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.pdf")
	require.NoError(t, os.WriteFile(path, []byte("pdf"), 0600))

	_, err := actionmessage.Send(context.Background(), actionmessage.SendRequest{
		RawRef: "@alice",
		Files:  []string{path, path},
		Names:  []string{"only-one"},
	}, func(context.Context, actionmessage.SendQuery) ([]output.SendResultRow, error) {
		t.Fatal("send function must not run for mismatched names")
		return nil, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestDownload_NormalizesMessageRef(t *testing.T) {
	row, err := actionmessage.Download(context.Background(), actionmessage.DownloadRequest{
		RawMessageRef: "@alice:42",
		Output:        " out.jpg ",
	}, func(_ context.Context, q actionmessage.DownloadQuery) (output.DownloadRow, error) {
		require.Equal(t, "alice", q.Ref.Value)
		require.Equal(t, 42, q.Message)
		require.Equal(t, "out.jpg", q.Output)
		return output.DownloadRow{MessageRef: "@alice:42", Path: "out.jpg"}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "out.jpg", row.Path)
}

func TestDownload_RequiresMessageRef(t *testing.T) {
	_, err := actionmessage.Download(context.Background(), actionmessage.DownloadRequest{
		RawMessageRef: "@alice",
	}, func(context.Context, actionmessage.DownloadQuery) (output.DownloadRow, error) {
		t.Fatal("download function must not run for invalid refs")
		return output.DownloadRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestDelete_ConfirmsBeforeDispatch(t *testing.T) {
	called := false
	result, err := actionmessage.Delete(context.Background(), actionmessage.DeleteRequest{
		RawMessageRefs: []string{"@chat:1", "@chat:2"},
		Prompter:       stubPrompter{ok: true},
	}, func(_ context.Context, q actionmessage.DeleteQuery) error {
		called = true
		require.Equal(t, []int{1, 2}, q.IDs)
		return nil
	})
	require.NoError(t, err)
	require.True(t, called)
	require.Equal(t, actionmessage.DeleteResult{Verb: "deleted", Count: 2}, result)
}

func TestDelete_DeclineSkipsDispatch(t *testing.T) {
	called := false
	_, err := actionmessage.Delete(context.Background(), actionmessage.DeleteRequest{
		RawMessageRefs: []string{"@chat:1"},
		Prompter:       stubPrompter{ok: false},
	}, func(context.Context, actionmessage.DeleteQuery) error {
		called = true
		return nil
	})
	require.ErrorIs(t, err, command.ErrNotConfirmed)
	require.False(t, called)
}

func TestReact_RequiresExactlyOneMode(t *testing.T) {
	_, err := actionmessage.React(context.Background(), actionmessage.ReactRequest{
		RawMessageRef: "@chat:1",
	}, func(context.Context, actionmessage.ReactQuery) (output.SendResultRow, error) {
		t.Fatal("react function must not run without a mode")
		return output.SendResultRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)

	_, err = actionmessage.React(context.Background(), actionmessage.ReactRequest{
		RawMessageRef: "@chat:1",
		Emoji:         "👍",
		Clear:         true,
	}, func(context.Context, actionmessage.ReactQuery) (output.SendResultRow, error) {
		t.Fatal("react function must not run with conflicting modes")
		return output.SendResultRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestLink_BuildsPublicAndPrivateURLs(t *testing.T) {
	publicURL, err := actionmessage.Link(context.Background(), actionmessage.LinkRequest{
		RawMessageRef: "@news:42",
	}, func(context.Context, actionmessage.LinkQuery) (actionmessage.LinkPeer, error) {
		return actionmessage.LinkPeer{Username: "news", IsChannel: true}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "https://t.me/news/42", publicURL)

	privateURL, err := actionmessage.Link(context.Background(), actionmessage.LinkRequest{
		RawMessageRef: "c:123:456:7",
	}, func(context.Context, actionmessage.LinkQuery) (actionmessage.LinkPeer, error) {
		return actionmessage.LinkPeer{ChannelID: 123, IsChannel: true}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "https://t.me/c/123/7", privateURL)
}

func TestLink_RejectsNonChannelPeer(t *testing.T) {
	_, err := actionmessage.Link(context.Background(), actionmessage.LinkRequest{
		RawMessageRef: "@alice:1",
	}, func(context.Context, actionmessage.LinkQuery) (actionmessage.LinkPeer, error) {
		return actionmessage.LinkPeer{Username: "alice"}, nil
	})
	require.ErrorIs(t, err, actionmessage.ErrNoLink)
}

type stubPrompter struct{ ok bool }

func (s stubPrompter) Confirm(string, bool) (bool, error)   { return s.ok, nil }
func (s stubPrompter) Password(string) (string, error)      { return "", nil }
func (s stubPrompter) Select(string, []string) (int, error) { return 0, nil }
func (s stubPrompter) Input(string, string) (string, error) { return "", nil }
