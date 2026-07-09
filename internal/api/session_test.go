package api

import (
	"testing"

	"github.com/arifaqyl/threadterm/internal/models"
)

func TestDedupePosts(t *testing.T) {
	in := []models.Post{
		{ID: "1", Text: "a"},
		{ID: "1", Text: "dup"},
		{ID: ""},
		{ID: "2", Text: "b"},
	}
	got := dedupePosts(in)
	if len(got) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(got))
	}
	if got[0].ID != "1" || got[1].ID != "2" {
		t.Fatalf("unexpected order/content: %+v", got)
	}
}

func TestFormatFollowers(t *testing.T) {
	cases := map[int]string{
		999:     "999",
		1_200:   "1.2K",
		2_500_0: "25.0K",
	}
	for n, want := range cases {
		if got := formatFollowers(n); got != want {
			t.Fatalf("formatFollowers(%d)=%s want %s", n, got, want)
		}
	}
}

func TestQueryTokens(t *testing.T) {
	got := queryTokens("LRT kelana-jaya line!!")
	want := []string{"lrt", "kelana", "jaya", "line"}
	if len(got) != len(want) {
		t.Fatalf("tokens len mismatch: got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("token[%d]=%q want %q", i, got[i], want[i])
		}
	}
}

func TestFilterPostsByQuery(t *testing.T) {
	posts := []models.Post{
		{ID: "1", Username: "a", Text: "lrt kelana jaya line delay"},
		{ID: "2", Username: "b", Text: "traffic in KL"},
		{ID: "3", Username: "kelana_updates", Text: "line issue"},
	}
	got := filterPostsByQuery(posts, "lrt kelana jaya")
	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("unexpected filter result: %+v", got)
	}
}

func TestFilterPostsByQuery_MultiTermNoisyFallbackBlocked(t *testing.T) {
	posts := []models.Post{
		{ID: "1", Username: "kelanaland", Text: "random text"},
		{ID: "2", Username: "lrtnews", Text: "another text"},
	}
	got := filterPostsByQuery(posts, "lrt kelana jaya")
	if len(got) != 0 {
		t.Fatalf("expected no results for noisy partial matches, got %+v", got)
	}
}

func TestParsePostsFromSearchHTML(t *testing.T) {
	html := `{"id":"3931234567890123456","username":"myrapidkl","text":"LRT Kelana Jaya line delay update"}`
	got := parsePostsFromSearchHTML(html)
	if len(got) != 1 {
		t.Fatalf("expected 1 post, got %d", len(got))
	}
	if got[0].Username != "myrapidkl" {
		t.Fatalf("unexpected username: %s", got[0].Username)
	}
	if got[0].ID != "3931234567890123456" {
		t.Fatalf("unexpected id: %s", got[0].ID)
	}
	if got[0].Text != "LRT Kelana Jaya line delay update" {
		t.Fatalf("unexpected text: %q", got[0].Text)
	}
}
