package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCookieHeader(t *testing.T) {
	got := ParseCookieHeader("sessionid=s1; csrftoken=c1; ds_user_id=42; mid=m1; ig_did=i1")
	if got.SessionID != "s1" || got.CSRFToken != "c1" || got.DSUserID != "42" || got.Mid != "m1" || got.IgDid != "i1" {
		t.Fatalf("unexpected parse result: %+v", got)
	}
}

func TestModePriority(t *testing.T) {
	cfg := &Config{}
	if cfg.Mode() != "demo" {
		t.Fatalf("expected demo, got %s", cfg.Mode())
	}

	cfg.Session = SessionCookies{SessionID: "s", CSRFToken: "c", DSUserID: "u"}
	if cfg.Mode() != "session" {
		t.Fatalf("expected session, got %s", cfg.Mode())
	}

	cfg.Bearer = BearerAuth{Token: "t", UserID: "u"}
	if cfg.Mode() != "live" {
		t.Fatalf("expected live, got %s", cfg.Mode())
	}

	cfg.Demo = true
	if cfg.Mode() != "demo" {
		t.Fatalf("expected demo when forced, got %s", cfg.Mode())
	}
}

func TestLoadErrorsOnCorruptJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOME", dir)

	cfgDir := filepath.Join(dir, ".threadterm")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte("{not valid json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("expected Load to error on corrupt JSON, got nil")
	}
}
