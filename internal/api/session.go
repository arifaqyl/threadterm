package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/arifaqyl/threadterm/internal/config"
	"github.com/arifaqyl/threadterm/internal/models"
	threads "github.com/teslashibe/threads-go"
)

// sessionClient talks to Threads via browser cookies (+ optional Bearer for writes).
// This is the primary live path — no Meta developer app required.
type sessionClient struct {
	cfg *config.Config
	tc  *threads.Client
}

func newSessionClient(cfg *config.Config) (*sessionClient, error) {
	cookies := threads.Cookies{
		SessionID: cfg.Session.SessionID,
		CSRFToken: cfg.Session.CSRFToken,
		DSUserID:  cfg.Session.DSUserID,
		Mid:       cfg.Session.Mid,
		IgDid:     cfg.Session.IgDid,
	}
	var tc *threads.Client
	var err error
	if cfg.HasBearer() {
		auth := threads.Auth{
			Token:    cfg.Bearer.Token,
			UserID:   cfg.Bearer.UserID,
			DeviceID: cfg.Bearer.DeviceID,
		}
		tc, err = threads.NewFull(cookies, auth, threads.WithMinRequestGap(1500*time.Millisecond))
	} else {
		tc, err = threads.New(cookies, threads.WithMinRequestGap(1500*time.Millisecond))
	}
	if err != nil {
		return nil, err
	}
	return &sessionClient{cfg: cfg, tc: tc}, nil
}

func (s *sessionClient) AuthStatus() models.AuthStatus {
	mode := s.cfg.Mode()
	return models.AuthStatus{
		Mode:     mode,
		Username: s.cfg.Username,
		UserID:   s.cfg.Session.DSUserID,
		Ready:    true,
	}
}

func (s *sessionClient) ensureUsername(ctx context.Context) {
	if s.cfg.Username != "" {
		return
	}
	me, err := s.tc.Me(ctx)
	if err != nil || me == nil {
		return
	}
	s.cfg.Username = me.Username
	s.cfg.UserID = me.ID
	_ = s.cfg.Save()
}

func (s *sessionClient) Feed(_ string, limit int) (models.FeedPage, error) {
	ctx := context.Background()
	s.ensureUsername(ctx)
	if limit <= 0 {
		limit = 25
	}

	// Prefer home timeline when we have write/bearer auth.
	if s.cfg.HasBearer() {
		page, err := s.tc.HomeTimeline(ctx, limit, "")
		if err == nil {
			return mapPostPage(page), nil
		}
		// fall through to own profile feed
	}

	uid := s.cfg.Session.DSUserID
	if uid == "" {
		me, err := s.tc.Me(ctx)
		if err != nil {
			return models.FeedPage{}, err
		}
		uid = me.ID
	}
	page, err := s.tc.UserThreads(ctx, uid, limit, "")
	if err != nil {
		return models.FeedPage{}, err
	}
	return mapPostPage(page), nil
}

func (s *sessionClient) Thread(id string) (models.Thread, error) {
	ctx := context.Background()
	tc, err := s.tc.GetThread(ctx, id)
	if err != nil {
		return models.Thread{}, err
	}
	root := firstPost(tc.ContainingThread)
	replies := flattenThreads(tc.ReplyThreads)
	return models.Thread{Root: root, Replies: replies}, nil
}

func (s *sessionClient) Profile(username string) (models.User, []models.Post, error) {
	ctx := context.Background()
	username = strings.TrimPrefix(username, "@")
	u, err := s.tc.GetProfileByUsername(ctx, username)
	if err != nil {
		return models.User{}, nil, err
	}
	page, err := s.tc.UserThreads(ctx, u.ID, 25, "")
	if err != nil {
		return mapUser(*u), nil, err
	}
	return mapUser(*u), flattenThreads(page.Threads), nil
}

func (s *sessionClient) Search(q string, limit int) (models.FeedPage, error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 25
	}
	// User search is what the private API exposes reliably.
	users, err := s.tc.SearchUsers(ctx, q, limit)
	if err != nil {
		return models.FeedPage{}, err
	}
	var posts []models.Post
	for _, u := range users.Users {
		posts = append(posts, models.Post{
			ID:       "user:" + u.ID,
			Username: u.Username,
			UserID:   u.ID,
			Text:     fmt.Sprintf("%s · %s followers", u.FullName, formatFollowers(u.FollowerCount)),
			Timestamp: time.Now().UTC(),
			MediaType: "USER",
		})
	}
	return models.FeedPage{Posts: posts}, nil
}

func (s *sessionClient) Publish(text string) (models.PublishResult, error) {
	if !s.cfg.HasBearer() {
		return models.PublishResult{}, fmt.Errorf("posting needs write login: threadterm login --user YOUR_USER --password … (or press a → write login in TUI)")
	}
	ctx := context.Background()
	post, err := s.tc.CreatePost(ctx, text)
	if err != nil {
		return models.PublishResult{}, err
	}
	return models.PublishResult{ID: post.ID, Permalink: post.Permalink}, nil
}

func (s *sessionClient) Reply(parentID, text string) (models.PublishResult, error) {
	if !s.cfg.HasBearer() {
		return models.PublishResult{}, fmt.Errorf("reply needs write login: threadterm login --user YOUR_USER --password …")
	}
	ctx := context.Background()
	post, err := s.tc.Reply(ctx, parentID, text)
	if err != nil {
		return models.PublishResult{}, err
	}
	return models.PublishResult{ID: post.ID, Permalink: post.Permalink}, nil
}

func (s *sessionClient) Like(id string) error {
	if !s.cfg.HasBearer() {
		return fmt.Errorf("like needs write login: threadterm login --user YOUR_USER --password …")
	}
	return s.tc.Like(context.Background(), id)
}

func (s *sessionClient) Unlike(id string) error {
	if !s.cfg.HasBearer() {
		return fmt.Errorf("unlike needs write login: threadterm login --user YOUR_USER --password …")
	}
	return s.tc.Unlike(context.Background(), id)
}

func mapPostPage(page threads.PostPage) models.FeedPage {
	return models.FeedPage{
		Posts:      flattenThreads(page.Threads),
		NextCursor: page.NextCursor,
	}
}

func flattenThreads(ths []threads.Thread) []models.Post {
	var out []models.Post
	for _, th := range ths {
		for _, p := range th.ThreadItems {
			out = append(out, mapPost(p))
		}
	}
	return out
}

func firstPost(th threads.Thread) models.Post {
	if len(th.ThreadItems) == 0 {
		return models.Post{}
	}
	return mapPost(th.ThreadItems[0])
}

func mapPost(p threads.Post) models.Post {
	media := "TEXT"
	switch p.MediaType {
	case 1:
		media = "IMAGE"
	case 2:
		media = "VIDEO"
	case 8:
		media = "CAROUSEL"
	}
	return models.Post{
		ID:          p.ID,
		Text:        p.Text,
		Username:    p.User.Username,
		UserID:      p.User.ID,
		Permalink:   p.Permalink,
		Timestamp:   p.TakenAt,
		LikeCount:   p.LikeCount,
		ReplyCount:  p.ReplyCount,
		RepostCount: p.RepostCount,
		MediaType:   media,
		LikedByMe:   p.HasLiked,
	}
}

func mapUser(u threads.User) models.User {
	return models.User{
		ID:        u.ID,
		Username:  u.Username,
		Name:      u.FullName,
		Bio:       u.Biography,
		Avatar:    u.ProfilePicURL,
		Verified:  u.IsVerified,
		Followers: u.FollowerCount,
		Following: u.FollowingCount,
		Threads:   u.MediaCount,
	}
}

func formatFollowers(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
