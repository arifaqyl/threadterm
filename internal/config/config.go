package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	EnvAccessToken = "THREADTERM_ACCESS_TOKEN"
	EnvUserID      = "THREADTERM_USER_ID"
	EnvClientID    = "THREADTERM_CLIENT_ID"
	EnvClientSecret = "THREADTERM_CLIENT_SECRET"
	EnvDemo        = "THREADTERM_DEMO"
)

// Config holds local threadterm settings.
type Config struct {
	AccessToken  string `json:"access_token,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	Username     string `json:"username,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	Demo         bool   `json:"demo,omitempty"`
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
	cfg := &Config{}
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, cfg)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

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
	if os.Getenv(EnvDemo) == "1" || os.Getenv(EnvDemo) == "true" {
		cfg.Demo = true
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

func (c *Config) HasToken() bool {
	return c.AccessToken != "" && c.UserID != ""
}

func (c *Config) Mode() string {
	if c.Demo || !c.HasToken() {
		return "demo"
	}
	return "token"
}
