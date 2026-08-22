package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	EnvAccessToken  = "THREADTERM_ACCESS_TOKEN"
	EnvUserID       = "THREADTERM_USER_ID"
	EnvClientID     = "THREADTERM_CLIENT_ID"
	EnvClientSecret = "THREADTERM_CLIENT_SECRET"
	EnvDemo         = "THREADTERM_DEMO"
	EnvTheme        = "THREADTERM_THEME"

	EnvDiscoverySeeds = "THREADTERM_DISCOVERY_SEEDS"

	EnvSessionID = "THREADS_SESSIONID"
	EnvCSRFToken = "THREADS_CSRFTOKEN"
	EnvDSUserID  = "THREADS_DS_USER_ID"
	EnvMid       = "THREADS_MID"
	EnvIgDid     = "THREADS_IG_DID"
	EnvBearer    = "THREADS_BEARER"
	EnvDeviceID  = "THREADS_DEVICE_ID"
)

// SessionCookies is the browser-session auth path (no Meta developer app).
type SessionCookies struct {
	SessionID string `json:"sessionid,omitempty"`
	CSRFToken string `json:"csrftoken,omitempty"`
	DSUserID  string `json:"ds_user_id,omitempty"`
	Mid       string `json:"mid,omitempty"`
	IgDid     string `json:"ig_did,omitempty"`
}

// BearerAuth is Bloks/Instagram write auth (post/like/reply).
type BearerAuth struct {
	Token    string `json:"token,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	DeviceID string `json:"device_id,omitempty"`
}

// Config holds local threadterm settings.
type Config struct {
	// Session = primary live path (browser cookies).
	Session SessionCookies `json:"session,omitempty"`
	Bearer  BearerAuth     `json:"bearer,omitempty"`

	// Official Graph API (optional / advanced).
	AccessToken  string `json:"access_token,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	Username     string `json:"username,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`

	Demo        bool   `json:"demo,omitempty"`
	Theme       string `json:"theme,omitempty"`
	SeenWelcome bool   `json:"seen_welcome,omitempty"`

	// DiscoverySeeds overrides the default public sample accounts used by
	// Discover / feed --discover. Comma-separated via THREADTERM_DISCOVERY_SEEDS.
	// Set it to "none" to disable seeding entirely.
	DiscoverySeeds []string `json:"discovery_seeds,omitempty"`
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".threadterm")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	cfg := &Config{Theme: "jade"}
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err == nil {
		if uerr := json.Unmarshal(data, cfg); uerr != nil {
			return nil, fmt.Errorf("parse config %s: %w", path, uerr)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	// Official API env
	if v := os.Getenv(EnvAccessToken); v != "" {
		cfg.AccessToken = v
	}
	if v := os.Getenv(EnvUserID); v != "" {
		cfg.UserID = v
	}
	if v := os.Getenv(EnvClientID); v != "" {
		cfg.ClientID = v
	}
	if v := os.Getenv(EnvClientSecret); v != "" {
		cfg.ClientSecret = v
	}
	if v := os.Getenv(EnvTheme); v != "" {
		cfg.Theme = v
	}
	if v := os.Getenv(EnvDiscoverySeeds); v != "" {
		cfg.DiscoverySeeds = ParseSeedList(v)
	}

	// Session env (Twitter-CLI style)
	if v := os.Getenv(EnvSessionID); v != "" {
		cfg.Session.SessionID = v
	}
	if v := os.Getenv(EnvCSRFToken); v != "" {
		cfg.Session.CSRFToken = v
	}
	if v := os.Getenv(EnvDSUserID); v != "" {
		cfg.Session.DSUserID = v
	}
	if v := os.Getenv(EnvMid); v != "" {
		cfg.Session.Mid = v
	}
	if v := os.Getenv(EnvIgDid); v != "" {
		cfg.Session.IgDid = v
	}
	if v := os.Getenv(EnvBearer); v != "" {
		cfg.Bearer.Token = v
	}
	if v := os.Getenv(EnvDeviceID); v != "" {
		cfg.Bearer.DeviceID = v
	}
	if os.Getenv(EnvDemo) == "1" || os.Getenv(EnvDemo) == "true" {
		cfg.Demo = true
	}
	if cfg.Theme == "" {
		cfg.Theme = "jade"
	}
	return cfg, nil
}

func (c *Config) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (c *Config) HasSession() bool {
	return c.Session.SessionID != "" && c.Session.CSRFToken != "" && c.Session.DSUserID != ""
}

func (c *Config) HasBearer() bool {
	return c.Bearer.Token != "" && c.Bearer.UserID != ""
}

func (c *Config) HasToken() bool {
	return c.AccessToken != "" && c.UserID != ""
}

// Mode priority: password/bearer > cookies > official token > demo
func (c *Config) Mode() string {
	if c.Demo {
		return "demo"
	}
	if c.HasBearer() {
		return "live"
	}
	if c.HasSession() {
		return "session"
	}
	if c.HasToken() {
		return "token"
	}
	return "demo"
}

func (c *Config) IsLive() bool {
	return !c.Demo && (c.HasBearer() || c.HasSession() || c.HasToken())
}

// ParseSeedList parses a comma-separated seed list. "none" (case-insensitive)
// returns an empty slice, which disables discovery seeding.
func ParseSeedList(raw string) []string {
	if strings.EqualFold(strings.TrimSpace(raw), "none") {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ParseCookieHeader parses a raw Cookie header or "name=value; name2=value2" paste.
func ParseCookieHeader(raw string) SessionCookies {
	out := SessionCookies{}
	raw = strings.TrimSpace(raw)
	parts := strings.Split(raw, ";")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(kv[0]))
		v := strings.TrimSpace(kv[1])
		switch k {
		case "sessionid":
			out.SessionID = v
		case "csrftoken":
			out.CSRFToken = v
		case "ds_user_id":
			out.DSUserID = v
		case "mid":
			out.Mid = v
		case "ig_did":
			out.IgDid = v
		}
	}
	return out
}
