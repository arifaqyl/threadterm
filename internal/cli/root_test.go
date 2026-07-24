package cli

import (
	"testing"

	"github.com/arifaqyl/threadterm/internal/api"
	"github.com/arifaqyl/threadterm/internal/config"
	"github.com/arifaqyl/threadterm/internal/models"
	"github.com/spf13/cobra"
)

// fakeClient records the backend calls each CLI command makes and returns
// canned, non-nil data so the command rendering paths never panic. It lets us
// assert CLI dispatch without touching the network or the on-disk config.
type fakeClient struct {
	calls []string
}

func (f *fakeClient) AuthStatus() models.AuthStatus {
	f.calls = append(f.calls, "authstatus")
	return models.AuthStatus{Mode: "demo", Ready: true, Username: "demo"}
}

func (f *fakeClient) Feed(cursor string, limit int) (models.FeedPage, error) {
	f.calls = append(f.calls, "feed")
	return models.FeedPage{Posts: []models.Post{{ID: "1", Username: "u", Text: "hi"}}}, nil
}

func (f *fakeClient) Discover(limit int) (models.FeedPage, error) {
	f.calls = append(f.calls, "discover")
	return models.FeedPage{Posts: []models.Post{{ID: "d1", Username: "u", Text: "d"}}}, nil
}

func (f *fakeClient) Thread(id string) (models.Thread, error) {
	f.calls = append(f.calls, "thread")
	return models.Thread{Root: models.Post{ID: id, Username: "u", Text: "root"}}, nil
}

func (f *fakeClient) Profile(username string) (models.User, []models.Post, error) {
	f.calls = append(f.calls, "profile")
	return models.User{Username: username}, []models.Post{{ID: "p1", Text: "x"}}, nil
}

func (f *fakeClient) Search(q string, limit int) (models.FeedPage, error) {
	f.calls = append(f.calls, "search")
	return models.FeedPage{Posts: []models.Post{{ID: "s1", Username: "u", Text: q}}}, nil
}

func (f *fakeClient) SearchUsers(q string, limit int) (models.FeedPage, error) {
	f.calls = append(f.calls, "searchusers")
	return models.FeedPage{Posts: []models.Post{{ID: "su1", Username: q}}}, nil
}

func (f *fakeClient) Latest(username string, limit int) (models.FeedPage, error) {
	f.calls = append(f.calls, "latest")
	return models.FeedPage{Posts: []models.Post{{ID: "l1", Username: username, Text: "x"}}}, nil
}

func (f *fakeClient) Publish(text string) (models.PublishResult, error) {
	f.calls = append(f.calls, "publish")
	return models.PublishResult{ID: "pub1"}, nil
}

func (f *fakeClient) Reply(parentID, text string) (models.PublishResult, error) {
	f.calls = append(f.calls, "reply")
	return models.PublishResult{ID: "rep1"}, nil
}

func (f *fakeClient) Like(id string) error {
	f.calls = append(f.calls, "like")
	return nil
}

func (f *fakeClient) Unlike(id string) error {
	f.calls = append(f.calls, "unlike")
	return nil
}

// setupFakeClient isolates the CLI from the real config (temp HOME) and
// injects a fake backend, then restores globals on cleanup.
func setupFakeClient(t *testing.T) *fakeClient {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOME", dir)
	demoForce = true
	jsonOut = false
	limit = 25
	fc := &fakeClient{}
	orig := clientFactory
	clientFactory = func(cfg *config.Config) api.Client { return fc }
	t.Cleanup(func() {
		clientFactory = orig
		demoForce = false
	})
	return fc
}

func TestCLISmokeDispatch(t *testing.T) {
	cases := []struct {
		name      string
		build     func() *cobra.Command
		args      []string
		flags     map[string]string
		wantCalls []string
	}{
		{"feed", cmdFeed, []string{}, nil, []string{"feed"}},
		{"feed_discover", cmdFeed, []string{}, map[string]string{"discover": "true"}, []string{"discover"}},
		{"search", cmdSearch, []string{"LRT", "KL"}, nil, []string{"search"}},
		{"search_users", cmdSearchUsers, []string{"myrapidkl"}, nil, []string{"searchusers"}},
		{"thread", cmdThread, []string{"abc"}, nil, []string{"thread"}},
		{"profile", cmdProfile, []string{"zuck"}, nil, []string{"profile"}},
		{"latest", cmdLatest, []string{"mosseri"}, nil, []string{"latest"}},
		{"like", cmdLike, []string{"abc"}, nil, []string{"like"}},
		{"unlike", cmdLike, []string{"abc"}, map[string]string{"unlike": "true"}, []string{"unlike"}},
		{"post", cmdPost, []string{"hi"}, nil, []string{"publish"}},
		{"whoami", cmdWhoami, []string{}, nil, []string{"authstatus"}},
		{"status", cmdStatus, []string{}, nil, []string{"authstatus", "feed"}},
		{"doctor", cmdDoctor, []string{}, nil, []string{"feed"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fc := setupFakeClient(t)
			cmd := tc.build()
			for k, v := range tc.flags {
				if err := cmd.Flags().Set(k, v); err != nil {
					t.Fatalf("set flag %s: %v", k, err)
				}
			}
			if err := cmd.RunE(cmd, tc.args); err != nil {
				t.Fatalf("RunE: %v", err)
			}
			if len(fc.calls) != len(tc.wantCalls) {
				t.Fatalf("calls=%v want=%v", fc.calls, tc.wantCalls)
			}
			for i, want := range tc.wantCalls {
				if fc.calls[i] != want {
					t.Fatalf("calls=%v want=%v", fc.calls, tc.wantCalls)
				}
			}
		})
	}
}
