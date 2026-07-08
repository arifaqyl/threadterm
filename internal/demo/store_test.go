package demo_test

import (
	"testing"

	"github.com/arifaqyl/threadterm/internal/demo"
)

func TestFeedAndPublish(t *testing.T) {
	s := demo.New()
	page, err := s.Feed("", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Posts) != 5 {
		t.Fatalf("want 5 posts, got %d", len(page.Posts))
	}
	res, err := s.Publish("test post from unit test")
	if err != nil {
		t.Fatal(err)
	}
	if res.ID == "" {
		t.Fatal("empty id")
	}
	page2, _ := s.Feed("", 1)
	if page2.Posts[0].Text != "test post from unit test" {
		t.Fatalf("publish did not prepend: %q", page2.Posts[0].Text)
	}
}

func TestSearchAndThread(t *testing.T) {
	s := demo.New()
	page, err := s.Search("threadterm", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Posts) == 0 {
		t.Fatal("expected search hits")
	}
	th, err := s.Thread(page.Posts[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(th.Replies) == 0 {
		t.Fatal("expected replies")
	}
}

func TestLike(t *testing.T) {
	s := demo.New()
	page, _ := s.Feed("", 1)
	id := page.Posts[0].ID
	before := page.Posts[0].LikeCount
	if err := s.Like(id); err != nil {
		t.Fatal(err)
	}
	page2, _ := s.Feed("", 1)
	if page2.Posts[0].LikeCount != before+1 || !page2.Posts[0].LikedByMe {
		t.Fatal("like did not apply")
	}
}
