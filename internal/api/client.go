package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/arifaqyl/threadterm/internal/config"
	"github.com/arifaqyl/threadterm/internal/demo"
	"github.com/arifaqyl/threadterm/internal/models"
)

const graphBase = "https://graph.threads.net/v1.0"

// Client is the Threads backend used by TUI and CLI.
type Client interface {
	AuthStatus() models.AuthStatus
	Feed(cursor string, limit int) (models.FeedPage, error)
	Thread(id string) (models.Thread, error)
	Profile(username string) (models.User, []models.Post, error)
	Search(q string, limit int) (models.FeedPage, error)
	Publish(text string) (models.PublishResult, error)
	Reply(parentID, text string) (models.PublishResult, error)
	Like(id string) error
	Unlike(id string) error
}

// New picks demo or live Graph API based on config.
func New(cfg *config.Config) Client {
	if cfg.Mode() == "demo" {
		return &demoAdapter{store: demo.New()}
	}
	return &graphClient{
		cfg:    cfg,
		http:   &http.Client{Timeout: 30 * time.Second},
		fields: "id,text,username,permalink,timestamp,media_type,media_url,shortcode,thumbnail_url,children,is_quote_post",
	}
}

type demoAdapter struct {
	store *demo.Store
}

func (d *demoAdapter) AuthStatus() models.AuthStatus { return d.store.AuthStatus() }
func (d *demoAdapter) Feed(c string, n int) (models.FeedPage, error) {
	return d.store.Feed(c, n)
}
func (d *demoAdapter) Thread(id string) (models.Thread, error) { return d.store.Thread(id) }
func (d *demoAdapter) Profile(u string) (models.User, []models.Post, error) {
	return d.store.Profile(u)
}
func (d *demoAdapter) Search(q string, n int) (models.FeedPage, error) {
	return d.store.Search(q, n)
}
func (d *demoAdapter) Publish(t string) (models.PublishResult, error) { return d.store.Publish(t) }
func (d *demoAdapter) Reply(p, t string) (models.PublishResult, error) {
	return d.store.Reply(p, t)
}
func (d *demoAdapter) Like(id string) error   { return d.store.Like(id) }
func (d *demoAdapter) Unlike(id string) error { return d.store.Unlike(id) }

type graphClient struct {
	cfg    *config.Config
	http   *http.Client
	fields string
}

func (g *graphClient) AuthStatus() models.AuthStatus {
	return models.AuthStatus{
		Mode:     "token",
		Username: g.cfg.Username,
		UserID:   g.cfg.UserID,
		Ready:    g.cfg.HasToken(),
	}
}

func (g *graphClient) get(path string, q url.Values, out any) error {
	if q == nil {
		q = url.Values{}
	}
	q.Set("access_token", g.cfg.AccessToken)
	u := graphBase + path + "?" + q.Encode()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := g.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("threads api %s: %s", resp.Status, truncate(string(body), 300))
	}
	return json.Unmarshal(body, out)
}

func (g *graphClient) postForm(path string, form url.Values, out any) error {
	if form == nil {
		form = url.Values{}
	}
	form.Set("access_token", g.cfg.AccessToken)
	u := graphBase + path
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, u, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := g.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("threads api %s: %s", resp.Status, truncate(string(body), 300))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

type mediaListResp struct {
	Data   []graphMedia `json:"data"`
	Paging *struct {
		Cursors struct {
			After string `json:"after"`
		} `json:"cursors"`
	} `json:"paging"`
}

type graphMedia struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	Username    string `json:"username"`
	Permalink   string `json:"permalink"`
	Timestamp   string `json:"timestamp"`
	MediaType   string `json:"media_type"`
	MediaURL    string `json:"media_url"`
	Shortcode   string `json:"shortcode"`
	IsQuotePost bool   `json:"is_quote_post"`
}

func (m graphMedia) toPost() models.Post {
	ts, _ := time.Parse(time.RFC3339, m.Timestamp)
	return models.Post{
		ID:        m.ID,
		Text:      m.Text,
		Username:  m.Username,
		Permalink: m.Permalink,
		Timestamp: ts,
		MediaType: m.MediaType,
		MediaURL:  m.MediaURL,
	}
}

func (g *graphClient) Feed(_ string, limit int) (models.FeedPage, error) {
	if limit <= 0 {
		limit = 25
	}
	q := url.Values{}
	q.Set("fields", g.fields)
	q.Set("limit", fmt.Sprintf("%d", limit))
	var resp mediaListResp
	if err := g.get("/"+g.cfg.UserID+"/threads", q, &resp); err != nil {
		return models.FeedPage{}, err
	}
	posts := make([]models.Post, 0, len(resp.Data))
	for _, m := range resp.Data {
		posts = append(posts, m.toPost())
	}
	page := models.FeedPage{Posts: posts}
	if resp.Paging != nil {
		page.NextCursor = resp.Paging.Cursors.After
	}
	return page, nil
}

func (g *graphClient) Thread(id string) (models.Thread, error) {
	q := url.Values{}
	q.Set("fields", g.fields)
	var m graphMedia
	if err := g.get("/"+id, q, &m); err != nil {
		return models.Thread{}, err
	}
	root := m.toPost()

	var replies mediaListResp
	rq := url.Values{}
	rq.Set("fields", g.fields)
	rq.Set("reverse", "false")
	_ = g.get("/"+id+"/replies", rq, &replies)

	out := make([]models.Post, 0, len(replies.Data))
	for _, r := range replies.Data {
		p := r.toPost()
		p.IsReply = true
		p.ReplyToID = id
		out = append(out, p)
	}
	return models.Thread{Root: root, Replies: out}, nil
}

func (g *graphClient) Profile(_ string) (models.User, []models.Post, error) {
	q := url.Values{}
	q.Set("fields", "id,username,name,threads_profile_discovery,threads_biography")
	var raw map[string]any
	if err := g.get("/me", q, &raw); err != nil {
		return models.User{}, nil, err
	}
	user := models.User{
		ID:       str(raw["id"]),
		Username: str(raw["username"]),
		Name:     str(raw["name"]),
		Bio:      str(raw["threads_biography"]),
	}
	feed, err := g.Feed("", 25)
	if err != nil {
		return user, nil, err
	}
	return user, feed.Posts, nil
}

func (g *graphClient) Search(query string, limit int) (models.FeedPage, error) {
	if limit <= 0 {
		limit = 25
	}
	q := url.Values{}
	q.Set("q", query)
	q.Set("search_type", "KEYWORD")
	q.Set("fields", g.fields)
	q.Set("limit", fmt.Sprintf("%d", limit))
	var resp mediaListResp
	if err := g.get("/keyword_search", q, &resp); err != nil {
		// Keyword search requires threads_keyword_search; fall back to local filter of own threads.
		feed, ferr := g.Feed("", 50)
		if ferr != nil {
			return models.FeedPage{}, err
		}
		ql := strings.ToLower(query)
		var hits []models.Post
		for _, p := range feed.Posts {
			if strings.Contains(strings.ToLower(p.Text), ql) || strings.Contains(strings.ToLower(p.Username), ql) {
				hits = append(hits, p)
			}
		}
		if limit > 0 && len(hits) > limit {
			hits = hits[:limit]
		}
		return models.FeedPage{Posts: hits}, nil
	}
	posts := make([]models.Post, 0, len(resp.Data))
	for _, m := range resp.Data {
		posts = append(posts, m.toPost())
	}
	return models.FeedPage{Posts: posts}, nil
}

func (g *graphClient) Publish(text string) (models.PublishResult, error) {
	form := url.Values{}
	form.Set("media_type", "TEXT")
	form.Set("text", text)
	var container struct {
		ID string `json:"id"`
	}
	if err := g.postForm("/"+g.cfg.UserID+"/threads", form, &container); err != nil {
		return models.PublishResult{}, err
	}
	// Give Meta a moment for TEXT containers (docs recommend wait for media).
	time.Sleep(500 * time.Millisecond)
	pub := url.Values{}
	pub.Set("creation_id", container.ID)
	var published struct {
		ID string `json:"id"`
	}
	if err := g.postForm("/"+g.cfg.UserID+"/threads_publish", pub, &published); err != nil {
		return models.PublishResult{}, err
	}
	permalink := ""
	th, err := g.Thread(published.ID)
	if err == nil {
		permalink = th.Root.Permalink
	}
	return models.PublishResult{ID: published.ID, Permalink: permalink, Container: container.ID}, nil
}

func (g *graphClient) Reply(parentID, text string) (models.PublishResult, error) {
	form := url.Values{}
	form.Set("media_type", "TEXT")
	form.Set("text", text)
	form.Set("reply_to_id", parentID)
	var container struct {
		ID string `json:"id"`
	}
	if err := g.postForm("/"+g.cfg.UserID+"/threads", form, &container); err != nil {
		return models.PublishResult{}, err
	}
	time.Sleep(500 * time.Millisecond)
	pub := url.Values{}
	pub.Set("creation_id", container.ID)
	var published struct {
		ID string `json:"id"`
	}
	if err := g.postForm("/"+g.cfg.UserID+"/threads_publish", pub, &published); err != nil {
		return models.PublishResult{}, err
	}
	return models.PublishResult{ID: published.ID, Container: container.ID}, nil
}

func (g *graphClient) Like(id string) error {
	return g.postForm("/"+id+"/like", nil, nil)
}

func (g *graphClient) Unlike(id string) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodDelete,
		fmt.Sprintf("%s/%s/like?access_token=%s", graphBase, id, url.QueryEscape(g.cfg.AccessToken)), nil)
	if err != nil {
		return err
	}
	resp, err := g.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("threads api %s: %s", resp.Status, truncate(string(body), 300))
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func str(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}
