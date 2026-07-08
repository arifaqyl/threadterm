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

// sessionClient talks to Threads via password (Bearer) and/or browser cookies.
type sessionClient struct {
	cfg *config.Config
	tc  *threads.Client
}

func newSessionClient(cfg *config.Config) (*sessionClient, error) {
	opts := []threads.Option{threads.WithMinRequestGap(1500 * time.Millisecond)}
	var tc *threads.Client
	var err error

	switch {
	case cfg.HasSession() && cfg.HasBearer():
		tc, err = threads.NewFull(
			toCookies(cfg),
			toAuth(cfg),
			opts...,
		)
	case cfg.HasBearer():
		// Normal username/password login — no cookie hustle.
		tc, err = threads.NewWithAuth(toAuth(cfg), opts...)
	case cfg.HasSession():
		tc, err = threads.New(toCookies(cfg), opts...)
	default:
		return nil, fmt.Errorf("no session or password login saved")
	}
	if err != nil {
		return nil, err
	}
	return &sessionClient{cfg: cfg, tc: tc}, nil
}

func toCookies(cfg *config.Config) threads.Cookies {
	return threads.Cookies{
		SessionID: cfg.Session.SessionID,
		CSRFToken: cfg.Session.CSRFToken,
		DSUserID:  cfg.Session.DSUserID,
		Mid:       cfg.Session.Mid,
		IgDid:     cfg.Session.IgDid,
	}
}

func toAuth(cfg *config.Config) threads.Auth {
	return threads.Auth{
		Token:    cfg.Bearer.Token,
		UserID:   cfg.Bearer.UserID,
		DeviceID: cfg.Bearer.DeviceID,
	}
}

func (s *sessionClient) AuthStatus() models.AuthStatus {
	uid := s.cfg.Bearer.UserID
	if uid == "" {
		uid = s.cfg.Session.DSUserID
	}
	return models.AuthStatus{
		Mode:     s.cfg.Mode(),
		Username: s.cfg.Username,
		UserID:   uid,
		Ready:    true,
	}
}

func (s *sessionClient) Feed(_ string, limit int) (models.FeedPage, error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 25
	}

	// Password login → home timeline (For You).
	if s.cfg.HasBearer() {
		page, err := s.tc.HomeTimeline(ctx, limit, "")
		if err == nil {
			return mapPostPage(page), nil
		}
		if !s.cfg.HasSession() {
			return models.FeedPage{}, fmt.Errorf("home feed failed: %w", err)
		}
		// fall through to profile feed if cookies exist
	}

	uid := s.cfg.Session.DSUserID
	if uid == "" {
		uid = s.cfg.Bearer.UserID
	}
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
	if !s.cfg.HasSession() {
		// Best-effort: some reply endpoints need cookies; still try.
	}
	tc, err := s.tc.GetThread(ctx, id)
	if err != nil {
		if !s.cfg.HasSession() {
			return models.Thread{}, fmt.Errorf("%w\n(tip: thread view works best after cookie login, but feed/post work with password alone)", err)
		}
		return models.Thread{}, err
	}
	root := firstPost(tc.ContainingThread)
	replies := flattenThreads(tc.ReplyThreads)
	return models.Thread{Root: root, Replies: replies}, nil
}

func (s *sessionClient) Profile(username string) (models.User, []models.Post, error) {
	ctx := context.Background()
	username = strings.TrimPrefix(username, "@")

	// Own profile without cookies.
	if s.cfg.HasBearer() && (username == "" || strings.EqualFold(username, s.cfg.Username) || username == "me") {
		u := models.User{
			ID:       s.cfg.Bearer.UserID,
			Username: s.cfg.Username,
		}
		page, err := s.tc.HomeTimeline(ctx, 25, "")
		if err != nil {
			return u, nil, nil
		}
		var mine []models.Post
		for _, p := range flattenThreads(page.Threads) {
			if strings.EqualFold(p.Username, s.cfg.Username) {
				mine = append(mine, p)
			}
		}
		return u, mine, nil
	}

	if !s.cfg.HasSession() {
		return models.User{}, nil, fmt.Errorf("looking up @%s needs browser cookies (optional). Password login already covers your feed + posting.\nRun: threadterm login --cookies \"…\"  OR stay on your feed", username)
	}

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
	if !s.cfg.HasSession() {
		// Bearer path: recommended users filtered locally, or clear error.
		users, err := s.tc.RecommendedUsers(ctx, limit)
		if err != nil {
			return models.FeedPage{}, fmt.Errorf("search needs cookies for full results; password login still gives home feed + post.\n%w", err)
		}
		ql := strings.ToLower(q)
		var posts []models.Post
		for _, u := range users.Users {
			if strings.Contains(strings.ToLower(u.Username), ql) || strings.Contains(strings.ToLower(u.FullName), ql) {
				posts = append(posts, models.Post{
					ID:        "user:" + u.ID,
					Username:  u.Username,
					UserID:    u.ID,
					Text:      fmt.Sprintf("%s · %s followers", u.FullName, formatFollowers(u.FollowerCount)),
					Timestamp: time.Now().UTC(),
					MediaType: "USER",
				})
			}
		}
		return models.FeedPage{Posts: posts}, nil
	}
	users, err := s.tc.SearchUsers(ctx, q, limit)
	if err != nil {
		return models.FeedPage{}, err
	}
	var posts []models.Post
	for _, u := range users.Users {
		posts = append(posts, models.Post{
			ID:        "user:" + u.ID,
			Username:  u.Username,
			UserID:    u.ID,
			Text:      fmt.Sprintf("%s · %s followers", u.FullName, formatFollowers(u.FollowerCount)),
			Timestamp: time.Now().UTC(),
			MediaType: "USER",
		})
	}
	return models.FeedPage{Posts: posts}, nil
}

func (s *sessionClient) Publish(text string) (models.PublishResult, error) {
	if !s.cfg.HasBearer() {
		return models.PublishResult{}, fmt.Errorf("not logged in — run: threadterm login")
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
		return models.PublishResult{}, fmt.Errorf("not logged in — run: threadterm login")
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
		return fmt.Errorf("not logged in — run: threadterm login")
	}
	return s.tc.Like(context.Background(), id)
}

func (s *sessionClient) Unlike(id string) error {
	if !s.cfg.HasBearer() {
		return fmt.Errorf("not logged in — run: threadterm login")
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
