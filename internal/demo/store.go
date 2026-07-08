package demo

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/arifaqyl/threadterm/internal/models"
)

// Store is an in-memory Threads-like dataset for offline / viral demos.
type Store struct {
	mu    sync.RWMutex
	posts []models.Post
	users map[string]models.User
	seq   int
}

func New() *Store {
	now := time.Now().UTC()
	s := &Store{
		users: map[string]models.User{
			"1": {ID: "1", Username: "zuck", Name: "Mark Zuckerberg", Verified: true, Followers: 9_200_000, Bio: "Building Threads."},
			"2": {ID: "2", Username: "arifaqyl", Name: "Arif Aqyl", Verified: false, Followers: 4200, Bio: "Building in public from MY 🇲🇾"},
			"3": {ID: "3", Username: "terminal_girl", Name: "Terminal Girl", Verified: true, Followers: 88000, Bio: "TUIs > apps"},
			"4": {ID: "4", Username: "golang", Name: "The Go Team", Verified: true, Followers: 510000, Bio: "Official Go"},
			"5": {ID: "5", Username: "mindofaqyl", Name: "mindofaqyl", Verified: false, Followers: 1200, Bio: "traffic + threads"},
		},
		posts: []models.Post{
			{
				ID: "p1", Text: "hot take: the best social client is still a terminal",
				Username: "terminal_girl", UserID: "3", Timestamp: now.Add(-12 * time.Minute),
				LikeCount: 842, ReplyCount: 63, RepostCount: 119, MediaType: "TEXT", Permalink: "https://www.threads.net/@terminal_girl/post/p1",
			},
			{
				ID: "p2", Text: "threadterm just dropped — Threads in your terminal.\n\nfeed · thread · compose · --json for agents\n\nbrew install soon™",
				Username: "arifaqyl", UserID: "2", Timestamp: now.Add(-28 * time.Minute),
				LikeCount: 210, ReplyCount: 41, RepostCount: 55, MediaType: "TEXT", Permalink: "https://www.threads.net/@arifaqyl/post/p2", TopicTag: "threadterm",
			},
			{
				ID: "p3", Text: "Bubble Tea + Lip Gloss = unfair advantage for CLI virality",
				Username: "golang", UserID: "4", Timestamp: now.Add(-1 * time.Hour),
				LikeCount: 1502, ReplyCount: 88, RepostCount: 301, MediaType: "TEXT", Permalink: "https://www.threads.net/@golang/post/p3",
			},
			{
				ID: "p4", Text: "If your social app needs a phone, it's already lost half the power users.",
				Username: "mindofaqyl", UserID: "5", Timestamp: now.Add(-2 * time.Hour),
				LikeCount: 96, ReplyCount: 17, RepostCount: 12, MediaType: "TEXT", Permalink: "https://www.threads.net/@mindofaqyl/post/p4",
			},
			{
				ID: "p5", Text: "Shipping in public: Threads API + TUI hybrid. Demo mode works offline. Token mode uses Meta Graph.",
				Username: "arifaqyl", UserID: "2", Timestamp: now.Add(-3 * time.Hour),
				LikeCount: 77, ReplyCount: 9, RepostCount: 14, MediaType: "TEXT", Permalink: "https://www.threads.net/@arifaqyl/post/p5",
			},
			{
				ID: "p6", Text: "tut for Mastodon. lazygit for git. threadterm for Threads.",
				Username: "terminal_girl", UserID: "3", Timestamp: now.Add(-5 * time.Hour),
				LikeCount: 1904, ReplyCount: 142, RepostCount: 410, MediaType: "TEXT", Permalink: "https://www.threads.net/@terminal_girl/post/p6",
			},
			{
				ID: "p7", Text: "Malaysia transit signals live at arifaqyl.me/traffic — next up: threadterm.",
				Username: "mindofaqyl", UserID: "5", Timestamp: now.Add(-8 * time.Hour),
				LikeCount: 54, ReplyCount: 6, RepostCount: 8, MediaType: "TEXT", Permalink: "https://www.threads.net/@mindofaqyl/post/p7",
			},
			{
				ID: "p8", Text: "Agents should post with `threadterm post \"hello\" --json` not a browser.",
				Username: "arifaqyl", UserID: "2", Timestamp: now.Add(-11 * time.Hour),
				LikeCount: 133, ReplyCount: 22, RepostCount: 31, MediaType: "TEXT", Permalink: "https://www.threads.net/@arifaqyl/post/p8",
			},
		},
		seq: 100,
	}
	return s
}

func (s *Store) AuthStatus() models.AuthStatus {
	return models.AuthStatus{Mode: "demo", Username: "demo", UserID: "2", Ready: true}
}

func (s *Store) Feed(_ string, limit int) (models.FeedPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.posts) {
		limit = len(s.posts)
	}
	out := make([]models.Post, limit)
	copy(out, s.posts[:limit])
	return models.FeedPage{Posts: out}, nil
}

func (s *Store) Thread(id string) (models.Thread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var root *models.Post
	for i := range s.posts {
		if s.posts[i].ID == id {
			root = &s.posts[i]
			break
		}
	}
	if root == nil {
		return models.Thread{}, fmt.Errorf("post not found: %s", id)
	}
	replies := []models.Post{
		{
			ID: id + "-r1", Text: "this is the way", Username: "golang", UserID: "4",
			Timestamp: root.Timestamp.Add(8 * time.Minute), LikeCount: 44, IsReply: true, ReplyToID: id, MediaType: "TEXT",
		},
		{
			ID: id + "-r2", Text: "installing rn", Username: "mindofaqyl", UserID: "5",
			Timestamp: root.Timestamp.Add(15 * time.Minute), LikeCount: 12, IsReply: true, ReplyToID: id, MediaType: "TEXT",
		},
		{
			ID: id + "-r3", Text: "need the VHS demo gif for X", Username: "terminal_girl", UserID: "3",
			Timestamp: root.Timestamp.Add(22 * time.Minute), LikeCount: 28, IsReply: true, ReplyToID: id, MediaType: "TEXT",
		},
	}
	return models.Thread{Root: *root, Replies: replies}, nil
}

func (s *Store) Profile(username string) (models.User, []models.Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	username = strings.TrimPrefix(strings.ToLower(username), "@")
	var user models.User
	found := false
	for _, u := range s.users {
		if strings.EqualFold(u.Username, username) {
			user = u
			found = true
			break
		}
	}
	if !found {
		return models.User{}, nil, fmt.Errorf("user not found: @%s", username)
	}
	var posts []models.Post
	for _, p := range s.posts {
		if strings.EqualFold(p.Username, username) {
			posts = append(posts, p)
		}
	}
	return user, posts, nil
}

func (s *Store) Search(q string, limit int) (models.FeedPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q = strings.ToLower(strings.TrimSpace(q))
	var hits []models.Post
	for _, p := range s.posts {
		if strings.Contains(strings.ToLower(p.Text), q) || strings.Contains(strings.ToLower(p.Username), q) || strings.Contains(strings.ToLower(p.TopicTag), q) {
			hits = append(hits, p)
		}
	}
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return models.FeedPage{Posts: hits}, nil
}

func (s *Store) Publish(text string) (models.PublishResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	id := fmt.Sprintf("p%d", s.seq)
	post := models.Post{
		ID: id, Text: text, Username: "arifaqyl", UserID: "2",
		Timestamp: time.Now().UTC(), MediaType: "TEXT",
		Permalink: "https://www.threads.net/@arifaqyl/post/" + id,
	}
	s.posts = append([]models.Post{post}, s.posts...)
	return models.PublishResult{ID: id, Permalink: post.Permalink, Container: "demo-" + id}, nil
}

func (s *Store) Reply(parentID, text string) (models.PublishResult, error) {
	res, err := s.Publish(text)
	if err != nil {
		return res, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.posts {
		if s.posts[i].ID == res.ID {
			s.posts[i].IsReply = true
			s.posts[i].ReplyToID = parentID
			break
		}
	}
	return res, nil
}

func (s *Store) Like(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.posts {
		if s.posts[i].ID == id {
			if !s.posts[i].LikedByMe {
				s.posts[i].LikedByMe = true
				s.posts[i].LikeCount++
			}
			return nil
		}
	}
	return fmt.Errorf("post not found: %s", id)
}

func (s *Store) Unlike(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.posts {
		if s.posts[i].ID == id {
			if s.posts[i].LikedByMe {
				s.posts[i].LikedByMe = false
				s.posts[i].LikeCount--
			}
			return nil
		}
	}
	return fmt.Errorf("post not found: %s", id)
}
