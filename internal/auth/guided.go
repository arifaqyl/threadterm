package auth

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/arifaqyl/threadterm/internal/config"
)

// GuidedCookieLogin walks the user through the easy browser-cookie path.
// This is the reliable login when Meta blocks Bloks password login.
func GuidedCookieLogin(cfg *config.Config) error {
	fmt.Println()
	fmt.Println("Meta blocked password login from this device")
	fmt.Println(`(their response: "Unable to log in — unexpected error")`)
	fmt.Println("That's NOT a wrong-password message — it's a checkpoint/risk block.")
	fmt.Println()
	fmt.Println("Easy fix — 30 seconds, same as Twitter CLIs:")
	fmt.Println()
	fmt.Println("  1) I'll open https://www.threads.com")
	fmt.Println("  2) Make sure you're logged in")
	fmt.Println("  3) Press F12 → Application → Cookies → www.threads.com")
	fmt.Println("  4) Copy the value of  sessionid")
	fmt.Println("     (also csrftoken + ds_user_id if you can — or paste all cookies)")
	fmt.Println()

	_ = openBrowser("https://www.threads.com")
	time.Sleep(800 * time.Millisecond)

	fmt.Print("Paste sessionid (or full cookie string), then Enter:\n> ")
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return fmt.Errorf("no input")
	}
	raw := strings.TrimSpace(sc.Text())
	if raw == "" {
		return fmt.Errorf("empty paste")
	}

	cookies := config.ParseCookieHeader(raw)
	// Allow pasting bare sessionid value
	if cookies.SessionID == "" && !strings.Contains(raw, "=") {
		cookies.SessionID = raw
	}
	if cookies.SessionID == "" {
		return fmt.Errorf("could not find sessionid in paste")
	}

	// Ask for missing required fields interactively
	if cookies.CSRFToken == "" {
		fmt.Print("csrftoken value: ")
		if sc.Scan() {
			cookies.CSRFToken = strings.TrimSpace(sc.Text())
		}
	}
	if cookies.DSUserID == "" {
		fmt.Print("ds_user_id value: ")
		if sc.Scan() {
			cookies.DSUserID = strings.TrimSpace(sc.Text())
		}
	}
	if cookies.Mid == "" {
		fmt.Print("mid value (optional, Enter to skip): ")
		if sc.Scan() {
			cookies.Mid = strings.TrimSpace(sc.Text())
		}
	}
	if cookies.IgDid == "" {
		fmt.Print("ig_did value (optional, Enter to skip): ")
		if sc.Scan() {
			cookies.IgDid = strings.TrimSpace(sc.Text())
		}
	}

	if cookies.CSRFToken == "" || cookies.DSUserID == "" {
		return fmt.Errorf("need csrftoken and ds_user_id too — copy them from the same Cookies panel")
	}

	if err := SetSession(cfg, cookies); err != nil {
		return err
	}
	fmt.Printf("\nlogged in as @%s · mode=%s\n", cfg.Username, cfg.Mode())
	fmt.Println("try:  threadterm")
	fmt.Println("      threadterm feed")
	return nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// ExplainLoginFailure turns opaque Bloks errors into actionable text.
func ExplainLoginFailure(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "no bearer token"),
		strings.Contains(lower, "unable to log in"),
		strings.Contains(lower, "unexpected error"):
		return fmt.Errorf(`Meta blocked password login from this device (not a wrong-password error).

What happened: Instagram returned "Unable to log in / unexpected error"
instead of a session token. Common causes: new device/IP, risk check,
or Bloks login being locked down.

Do this instead (works reliably):

  threadterm login --cookies

Or paste cookies after password fails when prompted.

How to get cookies:
  1. Open https://www.threads.com (logged in)
  2. F12 → Application → Cookies
  3. Copy sessionid, csrftoken, ds_user_id

SECURITY: if you typed your password in chat, change it now.`)
	case strings.Contains(lower, "two-factor"), strings.Contains(lower, "two_factor"):
		return fmt.Errorf("%w\n\nYour account has 2FA. Re-run with --totp YOUR_AUTHENTICATOR_SECRET", err)
	default:
		return err
	}
}
