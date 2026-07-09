package config

import "testing"

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
