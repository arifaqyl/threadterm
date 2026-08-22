package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/arifaqyl/threadterm/internal/api"
	"github.com/arifaqyl/threadterm/internal/auth"
	"github.com/arifaqyl/threadterm/internal/config"
	"github.com/arifaqyl/threadterm/internal/models"
	"github.com/arifaqyl/threadterm/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	jsonOut   bool
	demoForce bool
	limit     int
)

// Version is injected at build time via -ldflags.
var Version = "dev"

// clientFactory builds the api.Client from a config. Overridable in tests
// (default: api.New) so CLI commands can be exercised against a fake client
// without touching the network or the on-disk config.
var clientFactory = api.New

func Execute() error {
	root := &cobra.Command{
		Use:   "threadterm",
		Short: "Threads in your terminal - TUI + CLI",
		Long: `threadterm is a hybrid Threads client.

  threadterm                 open the TUI
  threadterm feed --json     agent-friendly feed
  threadterm post "hi"       publish from the shell
  threadterm login           auto from Chrome (like bird)

Default mode is demo (offline). Set a Meta Threads token for live.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, client, err := makeClient()
			if err != nil {
				return err
			}
			return tui.Run(cfg, client)
		},
	}

	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "emit JSON (for agents / scripts)")
	root.PersistentFlags().BoolVar(&demoForce, "demo", false, "force demo mode")
	root.PersistentFlags().IntVarP(&limit, "limit", "n", 25, "max items")

	root.AddCommand(
		cmdFeed(),
		cmdPost(),
		cmdReply(),
		cmdThread(),
		cmdSearch(),
		cmdSearchUsers(),
		cmdLatest(),
		cmdProfile(),
		cmdLike(),
		cmdLogin(),
		cmdLogout(),
		cmdWhoami(),
		cmdStatus(),
		cmdTheme(),
		cmdDoctor(),
		cmdVersion(),
	)

	return root.Execute()
}

func makeClient() (*config.Config, api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	if demoForce {
		cfg.Demo = true
	}
	return cfg, clientFactory(cfg), nil
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func cmdFeed() *cobra.Command {
	var discover bool
	cmd := &cobra.Command{
		Use:   "feed",
		Short: "Your following feed (or home timeline with write login)",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, c, err := makeClient()
			if err != nil {
				return err
			}
			var page models.FeedPage
			if discover {
				page, err = c.Discover(limit)
			} else {
				page, err = c.Feed("", limit)
			}
			if err != nil {
				return err
			}
			if jsonOut {
				return printJSON(page)
			}
			if page.Hint != "" && len(page.Posts) == 0 {
				fmt.Println(page.Hint)
				return nil
			}
			if len(page.Posts) == 0 {
				fmt.Println("no posts — try: threadterm search malaysia  or  threadterm feed --discover")
				return nil
			}
			if page.Source != "" {
				fmt.Printf("# source=%s", page.Source)
				if page.Hint != "" {
					fmt.Printf(" · %s", page.Hint)
				}
				fmt.Println()
			}
			for _, p := range page.Posts {
				fmt.Printf("%s  @%s  ♥%d\n  %s\n\n", p.ID, p.Username, p.LikeCount, indent(p.Text, 2))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&discover, "discover", false, "public sample feed (zuck/threads/…) — not your following")
	return cmd
}

func cmdPost() *cobra.Command {
	return &cobra.Command{
		Use:   "post [text]",
		Short: "Publish a text thread",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, c, err := makeClient()
			if err != nil {
				return err
			}
			text := strings.Join(args, " ")
			res, err := c.Publish(text)
			if err != nil {
				return err
			}
			if jsonOut {
				return printJSON(res)
			}
			fmt.Printf("posted %s\n", res.ID)
			if res.Permalink != "" {
				fmt.Println(res.Permalink)
			}
			return nil
		},
	}
}

func cmdReply() *cobra.Command {
	return &cobra.Command{
		Use:   "reply [post-id] [text]",
		Short: "Reply to a thread by id (same as the TUI R key)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, c, err := makeClient()
			if err != nil {
				return err
			}
			res, err := c.Reply(args[0], strings.Join(args[1:], " "))
			if err != nil {
				return err
			}
			if jsonOut {
				return printJSON(res)
			}
			fmt.Printf("replied %s\n", res.ID)
			if res.Permalink != "" {
				fmt.Println(res.Permalink)
			}
			return nil
		},
	}
}

func cmdThread() *cobra.Command {
	return &cobra.Command{
		Use:   "thread [id]",
		Short: "Show a thread and its replies",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, c, err := makeClient()
			if err != nil {
				return err
			}
			th, err := c.Thread(args[0])
			if err != nil {
				return err
			}
			if jsonOut {
				return printJSON(th)
			}
			fmt.Printf("@%s\n%s\n\n", th.Root.Username, th.Root.Text)
			for _, r := range th.Replies {
				fmt.Printf("  -> @%s: %s\n", r.Username, r.Text)
			}
			return nil
		},
	}
}

func cmdSearch() *cobra.Command {
	return &cobra.Command{
		Use:   "search [query]",
		Short: "Search posts (best-effort, source-labeled)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, c, err := makeClient()
			if err != nil {
				return err
			}
			page, err := c.Search(strings.Join(args, " "), limit)
			if err != nil {
				return err
			}
			if jsonOut {
				return printJSON(page)
			}
			if len(page.Posts) == 0 {
				fmt.Println("no hits")
				return nil
			}
			if page.Source != "" {
				fmt.Printf("# source=%s", page.Source)
				if page.Hint != "" {
					fmt.Printf(" · %s", page.Hint)
				}
				fmt.Println()
			}
			for _, p := range page.Posts {
				fmt.Printf("@%s  %s  ♥%d\n  %s\n\n", p.Username, p.ID, p.LikeCount, indent(p.Text, 2))
			}
			return nil
		},
	}
}

func cmdSearchUsers() *cobra.Command {
	return &cobra.Command{
		Use:   "search-users [query]",
		Short: "Search users/accounts by name or handle",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, c, err := makeClient()
			if err != nil {
				return err
			}
			page, err := c.SearchUsers(strings.Join(args, " "), limit)
			if err != nil {
				return err
			}
			if jsonOut {
				return printJSON(page)
			}
			if len(page.Posts) == 0 {
				if page.Hint != "" {
					fmt.Println(page.Hint)
				} else {
					fmt.Println("no users found")
				}
				return nil
			}
			if page.Source != "" {
				fmt.Printf("# source=%s", page.Source)
				if page.Hint != "" {
					fmt.Printf(" · %s", page.Hint)
				}
				fmt.Println()
			}
			for _, p := range page.Posts {
				fmt.Printf("@%s  %s\n", p.Username, p.ID)
				if p.Text != "" {
					fmt.Printf("  %s\n\n", p.Text)
				} else {
					fmt.Println()
				}
			}
			return nil
		},
	}
}

func cmdLatest() *cobra.Command {
	return &cobra.Command{
		Use:   "latest [username]",
		Short: "Latest posts from a user (watch / scrape helper)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, c, err := makeClient()
			if err != nil {
				return err
			}
			page, err := c.Latest(args[0], limit)
			if err != nil {
				return err
			}
			if jsonOut {
				return printJSON(page)
			}
			for _, p := range page.Posts {
				fmt.Printf("%s  @%s  ♥%d  %s\n  %s\n\n",
					p.Timestamp.Format(time.RFC3339), p.Username, p.LikeCount, p.ID, indent(p.Text, 2))
			}
			return nil
		},
	}
}

func cmdStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Quick auth + feed health check (like twitter status)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, c, err := makeClient()
			if err != nil {
				return err
			}
			st := c.AuthStatus()
			page, ferr := c.Feed("", 3)
			out := map[string]any{
				"mode":     st.Mode,
				"ready":    st.Ready,
				"username": st.Username,
				"user_id":  st.UserID,
				"theme":    cfg.Theme,
				"session":  cfg.HasSession(),
				"bearer":   cfg.HasBearer(),
				"feed_ok":  ferr == nil,
				"feed_n":   len(page.Posts),
			}
			if ferr != nil {
				out["feed_error"] = ferr.Error()
			}
			if jsonOut {
				return printJSON(out)
			}
			fmt.Printf("mode=%s user=@%s ready=%v session=%v bearer=%v feed=%d",
				st.Mode, st.Username, st.Ready, cfg.HasSession(), cfg.HasBearer(), len(page.Posts))
			if ferr != nil {
				fmt.Printf(" err=%v", ferr)
			}
			fmt.Println()
			return nil
		},
	}
}

func cmdProfile() *cobra.Command {
	return &cobra.Command{
		Use:   "profile [username]",
		Short: "Show a profile and recent posts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, c, err := makeClient()
			if err != nil {
				return err
			}
			u, posts, err := c.Profile(args[0])
			if err != nil {
				return err
			}
			if jsonOut {
				return printJSON(map[string]any{"user": u, "posts": posts})
			}
			fmt.Printf("@%s  %s\n%s\n\n", u.Username, u.Name, u.Bio)
			for _, p := range posts {
				fmt.Printf("  %s\n", p.Text)
			}
			return nil
		},
	}
}

func cmdLike() *cobra.Command {
	var unlike bool
	cmd := &cobra.Command{
		Use:   "like [id]",
		Short: "Like a post",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, c, err := makeClient()
			if err != nil {
				return err
			}
			if unlike {
				err = c.Unlike(args[0])
			} else {
				err = c.Like(args[0])
			}
			if err != nil {
				return err
			}
			if jsonOut {
				return printJSON(map[string]any{"ok": true, "id": args[0], "unlike": unlike})
			}
			if unlike {
				fmt.Println("unliked", args[0])
			} else {
				fmt.Println("liked", args[0])
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&unlike, "unlike", false, "remove like")
	return cmd
}

func cmdLogin() *cobra.Command {
	var (
		token, userID string
		cookieString  string
		sessionID     string
		csrf          string
		dsUser        string
		mid, igDid    string
		user, pass    string
		totp          string
		manualCookies bool
		usePassword   bool
		noBrowser     bool
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in (auto from Chrome - like bird/Twitter CLI)",
		Long: `Default (easiest - same as bird for Twitter):

  threadterm login

Reads your Threads session from Chrome/Edge/Brave/Firefox automatically.
Just be logged into https://www.threads.com in the browser first.

Other options:
  threadterm login --password-login --user X
  threadterm login --cookies      # guided manual cookie paste
  THREADTERM_PASSWORD=... threadterm login --password-login --user X`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if manualCookies {
				return auth.GuidedCookieLogin(cfg)
			}

			if cookieString != "" {
				if err := auth.SetSessionFromPaste(cfg, cookieString); err != nil {
					return err
				}
				fmt.Printf("cookies saved · @%s · mode=%s\n", cfg.Username, cfg.Mode())
				return nil
			}
			if sessionID != "" {
				sc := config.SessionCookies{
					SessionID: sessionID,
					CSRFToken: csrf,
					DSUserID:  dsUser,
					Mid:       mid,
					IgDid:     igDid,
				}
				if err := auth.SetSession(cfg, sc); err != nil {
					return err
				}
				fmt.Printf("cookies saved · @%s · mode=%s\n", cfg.Username, cfg.Mode())
				return nil
			}

			if token != "" {
				if userID == "" {
					return fmt.Errorf("--user-id required with --token")
				}
				if err := auth.SetToken(cfg, token, userID); err != nil {
					return err
				}
				fmt.Println("official token saved for", cfg.UserID, cfg.Username)
				return nil
			}

			// Explicit password path
			if usePassword || user != "" {
				if user == "" {
					user, err = promptLine("username: ")
					if err != nil {
						return err
					}
				}
				if pass == "" {
					pass = os.Getenv("THREADTERM_PASSWORD")
				}
				if pass == "" {
					pass, err = promptPassword("password: ")
					if err != nil {
						return err
					}
				}
				fmt.Println("logging in...")
				if err := auth.LoginPasswordTOTP(cfg, user, pass, totp); err != nil {
					fmt.Println()
					fmt.Println(auth.ExplainLoginFailure(err))
					fmt.Println()
					fmt.Println("Falling back to browser session (bird-style)...")
					if err2 := auth.LoginFromBrowser(cfg); err2 == nil {
						return nil
					}
					fmt.Print("Open guided cookie paste? [Y/n] ")
					ans, _ := promptLine("")
					ans = strings.ToLower(strings.TrimSpace(ans))
					if ans == "" || ans == "y" || ans == "yes" {
						return auth.GuidedCookieLogin(cfg)
					}
					return auth.ExplainLoginFailure(err)
				}
				fmt.Printf("logged in as @%s · mode=%s\n", cfg.Username, cfg.Mode())
				return nil
			}

			// DEFAULT: auto browser cookies (bird UX)
			if !noBrowser {
				if err := auth.LoginFromBrowser(cfg); err == nil {
					fmt.Println("try:  threadterm")
					fmt.Println("      threadterm feed")
					return nil
				} else {
					fmt.Println("browser auto-login:", err)
					fmt.Println()
					fmt.Println("Make sure you're logged into https://www.threads.com in Chrome, then retry.")
					fmt.Print("Open guided cookie paste instead? [Y/n] ")
					ans, _ := promptLine("")
					ans = strings.ToLower(strings.TrimSpace(ans))
					if ans == "" || ans == "y" || ans == "yes" {
						return auth.GuidedCookieLogin(cfg)
					}
					return err
				}
			}

			return fmt.Errorf("nothing to do - run: threadterm login")
		},
	}
	cmd.Flags().BoolVar(&manualCookies, "cookies", false, "guided manual cookie paste")
	cmd.Flags().BoolVar(&usePassword, "password-login", false, "use username+password instead of browser")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "skip auto browser cookie extraction")
	cmd.Flags().StringVar(&cookieString, "cookie-string", "", "raw Cookie header from threads.com")
	cmd.Flags().StringVar(&sessionID, "session-id", "", "sessionid cookie")
	cmd.Flags().StringVar(&csrf, "csrf", "", "csrftoken cookie")
	cmd.Flags().StringVar(&dsUser, "ds-user-id", "", "ds_user_id cookie")
	cmd.Flags().StringVar(&mid, "mid", "", "mid cookie")
	cmd.Flags().StringVar(&igDid, "ig-did", "", "ig_did cookie")
	cmd.Flags().StringVar(&user, "user", "", "Threads/IG username (password login)")
	cmd.Flags().StringVar(&totp, "totp", "", "authenticator 2FA secret")
	cmd.Flags().StringVar(&token, "token", "", "official Graph access token")
	cmd.Flags().StringVar(&userID, "user-id", "", "official Graph user id")
	return cmd
}
func cmdLogout() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear saved credentials (back to demo)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			theme := cfg.Theme
			seen := cfg.SeenWelcome
			cfg = &config.Config{Demo: true, Theme: theme, SeenWelcome: seen}
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Println("logged out · demo mode")
			return nil
		},
	}
}

func cmdWhoami() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show auth status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, c, err := makeClient()
			if err != nil {
				return err
			}
			st := c.AuthStatus()
			if jsonOut {
				return printJSON(map[string]any{"auth": st, "theme": cfg.Theme})
			}
			fmt.Printf("mode=%s ready=%v user=%s id=%s theme=%s\n", st.Mode, st.Ready, st.Username, st.UserID, cfg.Theme)
			return nil
		},
	}
}

func cmdTheme() *cobra.Command {
	return &cobra.Command{
		Use:   "theme [name]",
		Short: "Get or set TUI theme (jade|ocean|ember|mono|orchid)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			valid := map[string]bool{"jade": true, "ocean": true, "ember": true, "mono": true, "orchid": true}
			if len(args) == 0 {
				if jsonOut {
					return printJSON(map[string]any{"theme": cfg.Theme, "available": []string{"jade", "ocean", "ember", "mono", "orchid"}})
				}
				fmt.Println("current:", cfg.Theme)
				fmt.Println("available: jade · ocean · ember · mono · orchid")
				fmt.Println("tip: press t inside the TUI, or T to cycle")
				return nil
			}
			name := strings.ToLower(args[0])
			if !valid[name] {
				return fmt.Errorf("unknown theme %q (jade|ocean|ember|mono|orchid)", name)
			}
			cfg.Theme = name
			if err := cfg.Save(); err != nil {
				return err
			}
			if jsonOut {
				return printJSON(map[string]string{"theme": name})
			}
			fmt.Println("theme set to", name)
			return nil
		},
	}
}

func cmdDoctor() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check config, auth, and API reachability",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, c, err := makeClient()
			if err != nil {
				return err
			}
			path, _ := config.Path()
			fmt.Println("threadterm doctor")
			fmt.Println("  config:", path)
			fmt.Println("  mode:  ", cfg.Mode())
			fmt.Println("  theme: ", cfg.Theme)
			fmt.Println("  session:", boolStr(cfg.HasSession()))
			fmt.Println("  bearer: ", boolStr(cfg.HasBearer()))
			fmt.Println("  token: ", mask(cfg.AccessToken))
			fmt.Println("  user:  ", cfg.UserID, cfg.Username)
			fmt.Println("  app id:", mask(cfg.ClientID))
			page, err := c.Feed("", 3)
			if err != nil {
				fmt.Println("  feed:  FAIL", err)
				return err
			}
			fmt.Printf("  feed:  OK (%d posts)\n", len(page.Posts))
			return nil
		},
	}
}

func cmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(cmd *cobra.Command, args []string) error {
			v := map[string]string{"name": "threadterm", "version": Version}
			if jsonOut {
				return printJSON(v)
			}
			fmt.Println("threadterm", Version)
			return nil
		},
	}
}

func indent(s string, n int) string {
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i := range lines {
		if i == 0 {
			continue
		}
		lines[i] = pad + lines[i]
	}
	return strings.Join(lines, "\n")
}

func mask(s string) string {
	if s == "" {
		return "(empty)"
	}
	if len(s) < 8 {
		return "***"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

func boolStr(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func promptLine(label string) (string, error) {
	fmt.Fprint(os.Stdout, label)
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return "", err
		}
		return "", fmt.Errorf("no input")
	}
	return strings.TrimSpace(sc.Text()), nil
}

func promptPassword(label string) (string, error) {
	fmt.Fprint(os.Stdout, label)
	// Hidden input when stdin is a terminal; plain fallback otherwise.
	fd := int(syscall.Stdin)
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stdout)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return "", fmt.Errorf("no password")
	}
	return sc.Text(), nil
}
