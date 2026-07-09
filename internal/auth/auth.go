package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/arifaqyl/threadterm/internal/config"
	threads "github.com/teslashibe/threads-go"
)

const (
	authorizeURL  = "https://threads.net/oauth/authorize"
	tokenURL      = "https://graph.threads.net/oauth/access_token"
	longLivedURL  = "https://graph.threads.net/access_token"
	defaultScopes = "threads_basic,threads_content_publish,threads_read_replies,threads_manage_replies,threads_manage_insights,threads_keyword_search"
)

// SetSession saves browser cookies (primary live path — no Meta app).
func SetSession(cfg *config.Config, cookies config.SessionCookies) error {
	if cookies.SessionID == "" || cookies.CSRFToken == "" || cookies.DSUserID == "" {
		return fmt.Errorf("need sessionid, csrftoken, and ds_user_id (mid + ig_did recommended)")
	}
	cfg.Session = cookies
	cfg.Demo = false
	cfg.UserID = cookies.DSUserID

	// Best-effort username resolve
	tc, err := threads.New(threads.Cookies{
		SessionID: cookies.SessionID,
		CSRFToken: cookies.CSRFToken,
		DSUserID:  cookies.DSUserID,
		Mid:       cookies.Mid,
		IgDid:     cookies.IgDid,
	})
	if err == nil {
		if me, err := tc.Me(context.Background()); err == nil && me != nil {
			cfg.Username = me.Username
			cfg.UserID = me.ID
		}
	}
	return cfg.Save()
}

// SetSessionFromPaste accepts a raw Cookie header paste from DevTools.
func SetSessionFromPaste(cfg *config.Config, raw string) error {
	return SetSession(cfg, config.ParseCookieHeader(raw))
}

// LoginPassword is the normal path: username + password in the terminal.
// Uses Instagram Bloks login (same as the mobile app). Gives home feed,
// post, like, reply — no Meta developer app, no cookie hunting.
func LoginPassword(cfg *config.Config, username, password string) error {
	return LoginPasswordTOTP(cfg, username, password, "")
}

// LoginPasswordTOTP same as LoginPassword but auto-solves authenticator 2FA.
func LoginPasswordTOTP(cfg *config.Config, username, password, totpSecret string) error {
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	if username == "" || password == "" {
		return fmt.Errorf("username and password required")
	}
	deviceID := cfg.Bearer.DeviceID
	if deviceID == "" {
		deviceID = threads.GenerateDeviceID()
	}
	res, err := threads.LoginWith(context.Background(), threads.LoginParams{
		Username:   username,
		Password:   password,
		DeviceID:   deviceID,
		TOTPSecret: strings.TrimSpace(totpSecret),
	})
	if err != nil {
		return fmt.Errorf("login failed: %w\n\nIf your account has 2FA, run:\n  threadterm login --user YOU --password '…' --totp YOUR_AUTH_SECRET\nOr use authenticator app secret from Instagram 2FA settings.", err)
	}
	if res.Token == "" || res.UserID == "" {
		return fmt.Errorf("login returned empty credentials — Meta may have challenged this login (checkpoint). Try again later or from the same network you usually use Instagram")
	}
	cfg.Bearer = config.BearerAuth{
		Token:    res.Token,
		UserID:   res.UserID,
		DeviceID: deviceID,
	}
	cfg.Demo = false
	if res.Username != "" {
		cfg.Username = res.Username
	} else {
		cfg.Username = username
	}
	cfg.UserID = res.UserID
	return cfg.Save()
}

// ClearSession wipes session + bearer (back toward demo unless official token remains).
func ClearSession(cfg *config.Config) error {
	cfg.Session = config.SessionCookies{}
	cfg.Bearer = config.BearerAuth{}
	return cfg.Save()
}

// --- Official Graph OAuth (optional / advanced) ---

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

func ExchangeCode(clientID, clientSecret, redirectURI, code string) (accessToken string, userID string, err error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", redirectURI)
	form.Set("code", code)

	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
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
		return short.AccessToken, fmt.Sprintf("%d", short.UserID), nil
	}
	return long, fmt.Sprintf("%d", short.UserID), nil
}

func exchangeLongLived(clientSecret, shortToken string) (string, error) {
	q := url.Values{}
	q.Set("grant_type", "th_exchange_token")
	q.Set("client_secret", clientSecret)
	q.Set("access_token", shortToken)
	req, err := http.NewRequest(http.MethodGet, longLivedURL+"?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
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

func LoginLocalhost(cfg *config.Config, port int) (*config.Config, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("official OAuth needs THREADTERM_CLIENT_ID + CLIENT_SECRET (prefer cookie login instead)")
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
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Close() }()

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
		_ = enrichUsername(cfg)
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
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
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

func SetToken(cfg *config.Config, token, userID string) error {
	cfg.AccessToken = strings.TrimSpace(token)
	cfg.UserID = strings.TrimSpace(userID)
	cfg.Demo = false
	_ = enrichUsername(cfg)
	return cfg.Save()
}
