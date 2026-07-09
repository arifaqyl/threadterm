package api

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
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

// Opt-in discovery seeds (NOT your feed). Used only by Discover / feed --discover.
var discoverySeeds = []string{
	"zuck", "mosseri", "meta", "instagram", "threads",
	"golang", "openai", "anthropicai",
}

func newSessionClient(cfg *config.Config) (*sessionClient, error) {
	opts := []threads.Option{threads.WithMinRequestGap(800 * time.Millisecond)}
	var tc *threads.Client
	var err error

	switch {
	case cfg.HasSession() && cfg.HasBearer():
		tc, err = threads.NewFull(toCookies(cfg), toAuth(cfg), opts...)
	case cfg.HasBearer():
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

	// 1) Real home timeline when we have Bearer write auth.
	if s.cfg.HasBearer() {
		page, err := s.tc.HomeTimeline(ctx, limit, "")
		if err == nil && len(page.Threads) > 0 {
			out := mapPostPage(page)
			out.Source = "home"
			return out, nil
		}
	}

	// 2) Cookie session: ONLY people you follow (your feed), never seed spam.
	if s.cfg.HasSession() {
		posts, followingN, err := s.followingFeed(ctx, limit)
		if err != nil {
			return models.FeedPage{}, err
		}
		if len(posts) > 0 {
			return models.FeedPage{
				Posts:  posts,
				Source: "following",
				Hint:   fmt.Sprintf("%d accounts you follow", followingN),
			}, nil
		}
		hint := "you're not following anyone (or Threads hid the list) — press / to search, or: threadterm feed --discover"
		if followingN > 0 {
			hint = fmt.Sprintf("following %d accounts but no recent posts — press / to search", followingN)
		}
		return models.FeedPage{Posts: nil, Source: "empty", Hint: hint}, nil
	}

	// 3) Last resort: own profile threads.
	uid := s.cfg.Session.DSUserID
	if uid == "" {
		uid = s.cfg.Bearer.UserID
	}
	if uid != "" {
		page, err := s.tc.UserThreads(ctx, uid, limit, "")
		if err == nil {
			out := mapPostPage(page)
			out.Source = "profile"
			return out, nil
		}
	}
	return models.FeedPage{Posts: nil, Source: "empty", Hint: "no posts — try: threadterm search malaysia"}, nil
}

// Discover is an opt-in public sample feed (zuck/threads/etc). Not your timeline.
func (s *sessionClient) Discover(limit int) (models.FeedPage, error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 25
	}
	posts, err := s.seedFeed(ctx, discoverySeeds, limit)
	if err != nil {
		return models.FeedPage{}, err
	}
	return models.FeedPage{
		Posts:  posts,
		Source: "discover",
		Hint:   "public sample — not your following feed",
	}, nil
}

// followingFeed pulls recent posts from accounts you follow. No seeds.
func (s *sessionClient) followingFeed(ctx context.Context, limit int) ([]models.Post, int, error) {
	uid := s.cfg.Session.DSUserID
	if uid == "" {
		uid = s.cfg.Bearer.UserID
	}
	if uid == "" {
		return nil, 0, nil
	}

	following := map[string]string{}
	cursor := ""
	for pages := 0; pages < 3; pages++ {
		page, err := s.tc.GetFollowing(ctx, uid, 50, cursor)
		if err != nil {
			break
		}
		for _, u := range page.Users {
			following[strings.ToLower(u.Username)] = u.ID
		}
		if !page.HasNext || page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if len(following) == 0 {
		return nil, 0, nil
	}

	posts, err := s.postsFromUsers(ctx, following, 4, limit, 24)
	return posts, len(following), err
}

func (s *sessionClient) seedFeed(ctx context.Context, seeds []string, limit int) ([]models.Post, error) {
	users := map[string]string{}
	for _, seed := range seeds {
		users[strings.ToLower(seed)] = ""
	}
	return s.postsFromUsers(ctx, users, 3, limit, 12)
}

// postsFromUsers fetches recent threads for username→id map (id may be empty).
func (s *sessionClient) postsFromUsers(ctx context.Context, users map[string]string, perUser, limit, maxJobs int) ([]models.Post, error) {
	type job struct {
		name string
		id   string
	}
	jobs := make([]job, 0, len(users))
	for name, id := range users {
		jobs = append(jobs, job{name: name, id: id})
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].name < jobs[j].name })
	if maxJobs > 0 && len(jobs) > maxJobs {
		jobs = jobs[:maxJobs]
	}

	var (
		mu    sync.Mutex
		posts []models.Post
		wg    sync.WaitGroup
		sem   = make(chan struct{}, 4)
	)
	for _, j := range jobs {
		j := j
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			id := j.id
			if id == "" {
				u, err := s.tc.GetProfileByUsername(ctx, j.name)
				if err != nil || u == nil {
					return
				}
				id = u.ID
			}
			page, err := s.tc.UserThreads(ctx, id, perUser, "")
			if err != nil {
				return
			}
			got := flattenThreads(page.Threads)
			mu.Lock()
			posts = append(posts, got...)
			mu.Unlock()
		}()
	}
	wg.Wait()

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Timestamp.After(posts[j].Timestamp)
	})
	posts = dedupePosts(posts)
	if limit > 0 && len(posts) > limit {
		posts = posts[:limit]
	}
	return posts, nil
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
	if username == "" || strings.EqualFold(username, "me") {
		username = s.cfg.Username
	}
	if username == "" {
		return models.User{}, nil, fmt.Errorf("no username")
	}

	if !s.cfg.HasSession() && s.cfg.HasBearer() {
		u := models.User{ID: s.cfg.Bearer.UserID, Username: s.cfg.Username}
		return u, nil, nil
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

// Search finds users matching q, then returns their latest posts (agent-friendly).
func (s *sessionClient) Search(q string, limit int) (models.FeedPage, error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 25
	}
	q = strings.TrimSpace(q)
	if q == "" {
		return models.FeedPage{}, fmt.Errorf("empty query")
	}

	if !s.cfg.HasSession() {
		return models.FeedPage{}, fmt.Errorf("search needs browser session — run: threadterm login")
	}

	users, err := s.tc.SearchUsers(ctx, q, 12)
	if err != nil {
		return models.FeedPage{}, err
	}
	if len(users.Users) == 0 {
		return models.FeedPage{}, nil
	}

	var (
		mu    sync.Mutex
		posts []models.Post
		wg    sync.WaitGroup
		sem   = make(chan struct{}, 4)
	)
	maxUsers := len(users.Users)
	if maxUsers > 8 {
		maxUsers = 8
	}
	for i := 0; i < maxUsers; i++ {
		u := users.Users[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			page, err := s.tc.UserThreads(ctx, u.ID, 3, "")
			if err != nil {
				// still surface the user as a hit card
				mu.Lock()
				posts = append(posts, models.Post{
					ID:        "user:" + u.ID,
					Username:  u.Username,
					UserID:    u.ID,
					Text:      fmt.Sprintf("%s · %s followers · (no recent posts)", u.FullName, formatFollowers(u.FollowerCount)),
					Timestamp: time.Now().UTC(),
					MediaType: "USER",
				})
				mu.Unlock()
				return
			}
			got := flattenThreads(page.Threads)
			mu.Lock()
			posts = append(posts, got...)
			mu.Unlock()
		}()
	}
	wg.Wait()

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Timestamp.After(posts[j].Timestamp)
	})
	posts = dedupePosts(posts)
	if len(posts) > limit {
		posts = posts[:limit]
	}
	return models.FeedPage{Posts: posts, Source: "search", Hint: "users matching \"" + q + "\""}, nil
}

// Latest returns the newest posts from a username (scraping / watch helper).
func (s *sessionClient) Latest(username string, limit int) (models.FeedPage, error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 20
	}
	username = strings.TrimPrefix(username, "@")
	u, err := s.tc.GetProfileByUsername(ctx, username)
	if err != nil {
		return models.FeedPage{}, err
	}
	page, err := s.tc.UserThreads(ctx, u.ID, limit, "")
	if err != nil {
		return models.FeedPage{}, err
	}
	return mapPostPage(page), nil
}

func (s *sessionClient) Publish(text string) (models.PublishResult, error) {
	if !s.cfg.HasBearer() {
		return models.PublishResult{}, fmt.Errorf("posting needs write login: threadterm login --password-login")
	}
	post, err := s.tc.CreatePost(context.Background(), text)
	if err != nil {
		return models.PublishResult{}, err
	}
	return models.PublishResult{ID: post.ID, Permalink: post.Permalink}, nil
}

func (s *sessionClient) Reply(parentID, text string) (models.PublishResult, error) {
	if !s.cfg.HasBearer() {
		return models.PublishResult{}, fmt.Errorf("reply needs write login: threadterm login --password-login")
	}
	post, err := s.tc.Reply(context.Background(), parentID, text)
	if err != nil {
		return models.PublishResult{}, err
	}
	return models.PublishResult{ID: post.ID, Permalink: post.Permalink}, nil
}

func (s *sessionClient) Like(id string) error {
	if !s.cfg.HasBearer() {
		return fmt.Errorf("like needs write login: threadterm login --password-login")
	}
	return s.tc.Like(context.Background(), id)
}

func (s *sessionClient) Unlike(id string) error {
	if !s.cfg.HasBearer() {
		return fmt.Errorf("unlike needs write login: threadterm login --password-login")
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

func dedupePosts(in []models.Post) []models.Post {
	seen := map[string]bool{}
	var out []models.Post
	for _, p := range in {
		if p.ID == "" || seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		out = append(out, p)
	}
	return out
}
