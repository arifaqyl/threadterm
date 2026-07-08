package auth

import (
	"context"
	"fmt"

	"github.com/arifaqyl/threadterm/internal/browser"
	"github.com/arifaqyl/threadterm/internal/config"
)

// LoginFromBrowser auto-extracts Threads cookies from Chrome/Edge/Brave/Firefox
// — same UX as bird (Twitter CLI). No password. No DevTools paste.
func LoginFromBrowser(cfg *config.Config) error {
	fmt.Println("looking for Threads session in your browsers…")
	cookies, src, err := browser.ExtractThreadsSession(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("found session in", src)
	if err := SetSession(cfg, cookies); err != nil {
		return err
	}
	fmt.Printf("logged in as @%s · mode=%s\n", cfg.Username, cfg.Mode())
	return nil
}
