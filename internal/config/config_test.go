package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCookieHeader(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want SessionCookies
	}{
		{
			"all_five",
			"sessionid=s1; csrftoken=c1; ds_user_id=42; mid=m1; ig_did=i1",
			SessionCookies{SessionID: "s1", CSRFToken: "c1", DSUserID: "42", Mid: "m1", IgDid: "i1"},
		},
		{
			"required_only",
			"sessionid=s1; csrftoken=c1; ds_user_id=42",
			SessionCookies{SessionID: "s1", CSRFToken: "c1", DSUserID: "42"},
		},
		{
			"case_insensitive_keys",
			"SessionID=s1; CSRFToken=c1; DS_USER_ID=42",
			SessionCookies{SessionID: "s1", CSRFToken: "c1", DSUserID: "42"},
		},
		{
			"ignores_unknown_cookies",
			"foo=bar; sessionid=s1; csrftoken=c1; ds_user_id=42; baz=qux",
			SessionCookies{SessionID: "s1", CSRFToken: "c1", DSUserID: "42"},
		},
		{
			"skips_malformed_segments",
			"sessionid=s1; garbage-no-equals; csrftoken=c1; ds_user_id=42",
			SessionCookies{SessionID: "s1", CSRFToken: "c1", DSUserID: "42"},
		},
		{
			"trims_whitespace",
			"  sessionid = s1 ; csrftoken=c1 ; ds_user_id=42 ",
			SessionCookies{SessionID: "s1", CSRFToken: "c1", DSUserID: "42"},
		},
		{
			"empty_input",
			"   ",
			SessionCookies{},
		},
		{
			"only_unknown_cookies",
			"rando=keep; another=also",
			SessionCookies{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseCookieHeader(tc.raw)
			if got != tc.want {
				t.Fatalf("ParseCookieHeader(%q) = %+v, want %+v", tc.raw, got, tc.want)
			}
		})
	}
}

// Locks the trust boundary the bird-style login relies on: a parsed header needs
// all three of sessionid, csrftoken, ds_user_id for HasSession to be true.
func TestParseCookieHeaderHasSessionBoundary(t *testing.T) {
	segments := map[string]string{
		"sessionid":  "sessionid=s",
		"csrftoken":  "csrftoken=c",
		"ds_user_id": "ds_user_id=1",
	}
	order := []string{"sessionid", "csrftoken", "ds_user_id"}

	full := strings.Join([]string{segments["sessionid"], segments["csrftoken"], segments["ds_user_id"]}, "; ")
	if (&Config{Session: ParseCookieHeader(full)}).HasSession() != true {
		t.Fatalf("full header %q should satisfy HasSession", full)
	}
	for _, drop := range order {
		var kept []string
		for _, k := range order {
			if k == drop {
				continue
			}
			kept = append(kept, segments[k])
		}
		raw := strings.Join(kept, "; ")
		if (&Config{Session: ParseCookieHeader(raw)}).HasSession() {
			t.Fatalf("HasSession true after dropping %q (raw=%q)", drop, raw)
		}
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

func TestParseSeedList(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b ,c ", []string{"a", "b", "c"}},
		{"one", []string{"one"}},
		{"none", []string{}},
		{",,", []string{}},
		{"", []string{}},
	}
	for _, tc := range cases {
		got := ParseSeedList(tc.in)
		if len(got) != len(tc.want) {
			t.Fatalf("ParseSeedList(%q) = %v, want %v", tc.in, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("ParseSeedList(%q)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}

func TestLoadDiscoverySeedsFromEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOME", dir)
	t.Setenv(EnvDiscoverySeeds, "user1, user2")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.DiscoverySeeds) != 2 || cfg.DiscoverySeeds[0] != "user1" || cfg.DiscoverySeeds[1] != "user2" {
		t.Fatalf("expected env seeds [user1 user2], got %v", cfg.DiscoverySeeds)
	}

	t.Setenv(EnvDiscoverySeeds, "none")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.DiscoverySeeds) != 0 {
		t.Fatalf("expected empty seeds for 'none', got %v", cfg.DiscoverySeeds)
	}
}
