package browser_test

import (
	"context"
	"os"
	"testing"

	"github.com/arifaqyl/threadterm/internal/browser"
	"github.com/arifaqyl/threadterm/internal/config"
)

func TestSearchPostsLive(t *testing.T) {
	if testing.Short() || os.Getenv("THREADTERM_RUN_LIVE_TESTS") != "1" {
		t.Skip("set THREADTERM_RUN_LIVE_TESTS=1 to run live browser search test")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Skip(err)
	}
	if !cfg.HasSession() {
		t.Skip("no session")
	}
	posts, err := browser.SearchPosts(context.Background(), cfg, "lrt kelana jaya", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Fatal("expected posts")
	}
}
