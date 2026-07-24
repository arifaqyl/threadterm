package auth

import (
	"net/url"
	"testing"
)

func TestAuthURL(t *testing.T) {
	got := AuthURL("cid", "http://127.0.0.1:8765/callback", "state123")

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if u.Scheme != "https" || u.Host != "threads.net" || u.Path != "/oauth/authorize" {
		t.Fatalf("unexpected authorize URL: %s", got)
	}

	q := u.Query()
	if q.Get("client_id") != "cid" {
		t.Errorf("client_id=%q want cid", q.Get("client_id"))
	}
	if q.Get("redirect_uri") != "http://127.0.0.1:8765/callback" {
		t.Errorf("redirect_uri=%q want callback", q.Get("redirect_uri"))
	}
	if q.Get("response_type") != "code" {
		t.Errorf("response_type=%q want code", q.Get("response_type"))
	}
	if q.Get("state") != "state123" {
		t.Errorf("state=%q want state123", q.Get("state"))
	}
	if q.Get("scope") != defaultScopes {
		t.Errorf("scope=%q want %q", q.Get("scope"), defaultScopes)
	}
}

func TestAuthURLOmitsStateWhenEmpty(t *testing.T) {
	got := AuthURL("cid", "http://127.0.0.1:8765/callback", "")

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, ok := u.Query()["state"]; ok {
		t.Errorf("state should be omitted when empty, got %q", u.Query().Get("state"))
	}
}
