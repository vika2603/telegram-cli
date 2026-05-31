package discussion

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestRun_LinkHumanOutput(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &Options{
		RawChannel: "@chan",
		RawGroup:   "@grp",
		IOStreams:  ios,
		Do: func(_ context.Context, _ actionchat.DiscussionQuery) (output.DiscussionRow, error) {
			return output.DiscussionRow{
				Action:  "link",
				Channel: output.PeerRef{Username: "chan"},
				Group:   &output.PeerRef{Username: "grp"},
			}, nil
		},
	}
	require.NoError(t, Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "linked @chan → @grp")
}

func TestRun_UnlinkHumanOutput(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &Options{
		RawChannel: "@chan",
		Unlink:     true,
		IOStreams:  ios,
		Do: func(_ context.Context, _ actionchat.DiscussionQuery) (output.DiscussionRow, error) {
			return output.DiscussionRow{Action: "unlink", Channel: output.PeerRef{Username: "chan"}}, nil
		},
	}
	require.NoError(t, Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "unlinked discussion group from @chan")
}

func TestRunCandidates_HumanOutput(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &CandidatesOptions{
		IOStreams: ios,
		Fetch: func(context.Context) ([]output.ChatRow, error) {
			return []output.ChatRow{{ID: -100, Kind: "chat", Title: "FixtureGrp", Username: "fixturegrp"}}, nil
		},
	}
	require.NoError(t, runCandidates(context.Background(), opts))
	require.Contains(t, stdout.String(), "FixtureGrp")
}
