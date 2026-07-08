package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/arifaqyl/threadterm/internal/config"
)

const (
	authorizeURL = "https://threads.net/oauth/authorize"
	tokenURL     = "https://graph.threads.net/oauth/access_token"
	longLivedURL = "https://graph.threads.net/access_token"
	defaultScopes = "threads_basic,threads_content_publish,threads_read_replies,threads_manage_replies,threads_manage_insights,threads_keyword_search"
)

// AuthURL builds the Meta Threads OAuth authorize URL.
func AuthURL(clientID, redirectURI, state string) string {
	q := url.Values{}
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", defaultScopes)
	q.Set("response_type", "code")
	if state != "" {
		q.Set("state", state)
	}
	return authorizeURL + "?" + q.Encode()
}

// ExchangeCode swaps an auth code for a short-lived token, then upgrades to long-lived.
func ExchangeCode(clientID, clientSecret, redirectURI, code string) (accessToken string, userID string, err error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", redirectURI)
	form.Set("code", code)

	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("token exchange failed: %s", string(body))
	}
	var short struct {
		AccessToken string `json:"access_token"`
		UserID      int64  `json:"user_id"`
	}
	if err := json.Unmarshal(body, &short); err != nil {
		return "", "", err
	}
	long, err := exchangeLongLived(clientSecret, short.AccessToken)
	if err != nil {
		// Still usable short-lived.
		return short.AccessToken, fmt.Sprintf("%d", short.UserID), nil
	}
	return long, fmt.Sprintf("%d", short.UserID), nil
}

func exchangeLongLived(clientSecret, shortToken string) (string, error) {
	q := url.Values{}
	q.Set("grant_type", "th_exchange_token")
	q.Set("client_secret", clientSecret)
	q.Set("access_token", shortToken)
	resp, err := http.Get(longLivedURL + "?" + q.Encode())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("long-lived exchange failed: %s", string(body))
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	return out.AccessToken, nil
}

// LoginLocalhost runs a one-shot OAuth callback on 127.0.0.1 and saves config.
func LoginLocalhost(cfg *config.Config, port int) (*config.Config, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("set THREADTERM_CLIENT_ID and THREADTERM_CLIENT_SECRET (Meta Threads App)")
	}
	if port == 0 {
		port = 8765
	}
	redirect := fmt.Sprintf("http://127.0.0.1:%d/callback", port)
	state := fmt.Sprintf("tt-%d", time.Now().UnixNano())
	authURL := AuthURL(cfg.ClientID, redirect, state)

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, err
	}
	defer ln.Close()

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			errCh <- fmt.Errorf("oauth state mismatch")
			return
		}
		if e := r.URL.Query().Get("error"); e != "" {
			msg := r.URL.Query().Get("error_description")
			http.Error(w, msg, http.StatusBadRequest)
			errCh <- fmt.Errorf("%s: %s", e, msg)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			errCh <- fmt.Errorf("missing code")
			return
		}
		_, _ = io.WriteString(w, "<html><body><h1>threadterm</h1><p>Login OK. You can close this tab.</p></body></html>")
		codeCh <- code
	})
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	defer func() {
		_ = srv.Close()
	}()

	fmt.Println("Open this URL to authorize threadterm:")
	fmt.Println(authURL)
	fmt.Println()
	fmt.Println("Waiting for callback on", redirect, "…")

	select {
	case code := <-codeCh:
		token, uid, err := ExchangeCode(cfg.ClientID, cfg.ClientSecret, redirect, code)
		if err != nil {
			return nil, err
		}
		cfg.AccessToken = token
		cfg.UserID = uid
		cfg.Demo = false
		if err := enrichUsername(cfg); err != nil {
			// non-fatal
			_ = err
		}
		if err := cfg.Save(); err != nil {
			return nil, err
		}
		return cfg, nil
	case err := <-errCh:
		return nil, err
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("oauth timed out after 5m")
	}
}

func enrichUsername(cfg *config.Config) error {
	u := "https://graph.threads.net/v1.0/me?fields=id,username&access_token=" + url.QueryEscape(cfg.AccessToken)
	resp, err := http.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("me: %s", string(body))
	}
	var me struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}
	if err := json.Unmarshal(body, &me); err != nil {
		return err
	}
	if me.ID != "" {
		cfg.UserID = me.ID
	}
	cfg.Username = me.Username
	return nil
}

// SetToken saves a manually provided token + user id.
func SetToken(cfg *config.Config, token, userID string) error {
	cfg.AccessToken = strings.TrimSpace(token)
	cfg.UserID = strings.TrimSpace(userID)
	cfg.Demo = false
	_ = enrichUsername(cfg)
	return cfg.Save()
}
