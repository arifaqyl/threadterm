package browser

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/arifaqyl/threadterm/internal/config"
	"github.com/steipete/sweetcookie"
)

// ExtractThreadsSession reads Threads cookies from local browsers
// (Chrome / Edge / Brave / Firefox) — same idea as bird for Twitter.
func ExtractThreadsSession(ctx context.Context) (config.SessionCookies, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	names := []string{"sessionid", "csrftoken", "ds_user_id", "mid", "ig_did"}
	urls := []string{
		"https://www.threads.com/",
		"https://www.threads.net/",
		"https://threads.net/",
	}

	var (
		best     config.SessionCookies
		bestSrc  string
		bestHits int
		warns    []string
	)

	browsers := []sweetcookie.Browser{
		sweetcookie.BrowserChrome,
		sweetcookie.BrowserEdge,
		sweetcookie.BrowserBrave,
		sweetcookie.BrowserFirefox,
		sweetcookie.BrowserChromium,
	}

	for _, u := range urls {
		res, err := sweetcookie.Get(ctx, sweetcookie.Options{
			URL:      u,
			Names:    names,
			Browsers: browsers,
			Mode:     sweetcookie.ModeMerge,
		})
		if err != nil {
			warns = append(warns, err.Error())
			continue
		}
		for _, w := range res.Warnings {
			warns = append(warns, w)
		}

		byBrowser := map[string]config.SessionCookies{}
		srcName := map[string]string{}
		for _, c := range res.Cookies {
			key := string(c.Source.Browser) + "|" + c.Source.Profile
			sc := byBrowser[key]
			srcName[key] = fmt.Sprintf("%s (%s)", c.Source.Browser, c.Source.Profile)
			switch strings.ToLower(c.Name) {
			case "sessionid":
				if c.Value != "" {
					sc.SessionID = c.Value
				}
			case "csrftoken":
				if c.Value != "" {
					sc.CSRFToken = c.Value
				}
			case "ds_user_id":
				if c.Value != "" {
					sc.DSUserID = c.Value
				}
			case "mid":
				if c.Value != "" {
					sc.Mid = c.Value
				}
			case "ig_did":
				if c.Value != "" {
					sc.IgDid = c.Value
				}
			}
			byBrowser[key] = sc
		}

		for key, sc := range byBrowser {
			hits := 0
			if sc.SessionID != "" {
				hits++
			}
			if sc.CSRFToken != "" {
				hits++
			}
			if sc.DSUserID != "" {
				hits++
			}
			if sc.Mid != "" {
				hits++
			}
			if sc.IgDid != "" {
				hits++
			}
			// Prefer complete sessions.
			if sc.SessionID != "" && sc.CSRFToken != "" && sc.DSUserID != "" && hits > bestHits {
				best = sc
				bestSrc = srcName[key]
				bestHits = hits
			} else if bestHits < 3 && sc.SessionID != "" && hits > bestHits {
				best = sc
				bestSrc = srcName[key]
				bestHits = hits
			}
		}
	}

	if best.SessionID == "" || best.CSRFToken == "" || best.DSUserID == "" {
		msg := "no Threads session found in Chrome/Edge/Brave/Firefox"
		if len(warns) > 0 {
			msg += "\n" + strings.Join(uniq(warns), "\n")
		}
		msg += "\n\nFix: open https://www.threads.com in Chrome and log in, then run: threadterm login"
		return config.SessionCookies{}, "", fmt.Errorf("%s", msg)
	}
	return best, bestSrc, nil
}

func uniq(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, "  · "+s)
		if len(out) >= 8 {
			break
		}
	}
	return out
}
