package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/arifaqyl/threadterm/internal/api"
	"github.com/arifaqyl/threadterm/internal/auth"
	"github.com/arifaqyl/threadterm/internal/config"
	"github.com/arifaqyl/threadterm/internal/tui"
	"github.com/spf13/cobra"
)

var (
	jsonOut bool
	demoForce bool
	limit   int
)

func Execute() error {
	root := &cobra.Command{
		Use:   "threadterm",
		Short: "Threads in your terminal — TUI + CLI",
		Long: `threadterm is a hybrid Threads client.

  threadterm                 open the TUI
  threadterm feed --json     agent-friendly feed
  threadterm post "hi"       publish from the shell
  threadterm login           OAuth via localhost callback

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
		cmdThread(),
		cmdSearch(),
		cmdProfile(),
		cmdLike(),
		cmdLogin(),
		cmdLogout(),
		cmdWhoami(),
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
	return cfg, api.New(cfg), nil
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func cmdFeed() *cobra.Command {
	return &cobra.Command{
		Use:   "feed",
		Short: "Show your threads / demo feed",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, c, err := makeClient()
			if err != nil {
				return err
			}
			page, err := c.Feed("", limit)
			if err != nil {
				return err
			}
			if jsonOut {
				return printJSON(page)
			}
			for _, p := range page.Posts {
				fmt.Printf("%s  @%s  ♥%d\n  %s\n\n", p.ID, p.Username, p.LikeCount, indent(p.Text, 2))
			}
			return nil
		},
	}
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
				fmt.Printf("  └ @%s: %s\n", r.Username, r.Text)
			}
			return nil
		},
	}
}

func cmdSearch() *cobra.Command {
	return &cobra.Command{
		Use:   "search [query]",
		Short: "Search posts (keyword API or local filter)",
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
			for _, p := range page.Posts {
				fmt.Printf("@%s  %s\n  %s\n\n", p.Username, p.ID, indent(p.Text, 2))
			}
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
		port          int
		cookies       string
		sessionID     string
		csrf          string
		dsUser        string
		mid, igDid    string
		user, pass    string
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login via browser cookies (recommended) or password / official OAuth",
		Long: `Primary path (no Meta developer app):

  1. Open https://www.threads.com logged in
  2. DevTools → Application → Cookies → copy values
  3. threadterm login --cookies "sessionid=...; csrftoken=...; ds_user_id=...; mid=...; ig_did=..."

Or paste individually:
  threadterm login --session-id ... --csrf ... --ds-user-id ... --mid ... --ig-did ...

Write access (post/like/reply) — Bloks password login:
  threadterm login --user yourname --password '...'

Official Graph API (optional):
  threadterm login --token ... --user-id ...
  threadterm login   # OAuth if CLIENT_ID/SECRET set`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// Cookie paste (primary)
			if cookies != "" {
				if err := auth.SetSessionFromPaste(cfg, cookies); err != nil {
					return err
				}
				fmt.Printf("session saved · @%s (%s) · mode=%s\n", cfg.Username, cfg.UserID, cfg.Mode())
				fmt.Println("tip: for posting, also run: threadterm login --user YOU --password '…'")
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
				fmt.Printf("session saved · @%s (%s) · mode=%s\n", cfg.Username, cfg.UserID, cfg.Mode())
				return nil
			}

			// Password → bearer writes
			if user != "" {
				if pass == "" {
					return fmt.Errorf("--password required with --user")
				}
				if err := auth.LoginPassword(cfg, user, pass); err != nil {
					return err
				}
				fmt.Printf("write auth saved · @%s · mode=%s\n", cfg.Username, cfg.Mode())
				if !cfg.HasSession() {
					fmt.Println("note: add browser cookies for reading feeds (login --cookies …)")
				}
				return nil
			}

			// Official token
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

			// Official OAuth fallback
			cfg, err = auth.LoginLocalhost(cfg, port)
			if err != nil {
				return fmt.Errorf("%w\n\nprefer: threadterm login --cookies \"sessionid=…; csrftoken=…; ds_user_id=…\"", err)
			}
			fmt.Printf("logged in as @%s (%s)\n", cfg.Username, cfg.UserID)
			return nil
		},
	}
	cmd.Flags().StringVar(&cookies, "cookies", "", "raw Cookie header paste from threads.com DevTools")
	cmd.Flags().StringVar(&sessionID, "session-id", "", "sessionid cookie")
	cmd.Flags().StringVar(&csrf, "csrf", "", "csrftoken cookie")
	cmd.Flags().StringVar(&dsUser, "ds-user-id", "", "ds_user_id cookie")
	cmd.Flags().StringVar(&mid, "mid", "", "mid cookie")
	cmd.Flags().StringVar(&igDid, "ig-did", "", "ig_did cookie")
	cmd.Flags().StringVar(&user, "user", "", "Threads/IG username for write login")
	cmd.Flags().StringVar(&pass, "password", "", "password for write login (Bloks)")
	cmd.Flags().StringVar(&token, "token", "", "official Graph access token (optional)")
	cmd.Flags().StringVar(&userID, "user-id", "", "official Graph user id")
	cmd.Flags().IntVar(&port, "port", 8765, "localhost OAuth callback port")
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
			cfg, err := config.Load()
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
			c := api.New(cfg)
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
			v := map[string]string{"name": "threadterm", "version": "0.1.0"}
			if jsonOut {
				return printJSON(v)
			}
			fmt.Println("threadterm 0.1.0")
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
	return s[:4] + "…" + s[len(s)-4:]
}

func boolStr(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
