package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/arifaqyl/threadterm/internal/api"
	"github.com/arifaqyl/threadterm/internal/auth"
	"github.com/arifaqyl/threadterm/internal/config"
	"github.com/arifaqyl/threadterm/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	jsonOut bool
	demoForce bool
	limit   int
)

func Execute() error {
	root := &cobra.Command{
		Use:   "threadterm",
		Short: "Threads in your terminal â€” TUI + CLI",
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
				fmt.Printf("%s  @%s  â™¥%d\n  %s\n\n", p.ID, p.Username, p.LikeCount, indent(p.Text, 2))
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
				fmt.Printf("  â”” @%s: %s\n", r.Username, r.Text)
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
		cookieString  string
		sessionID     string
		csrf          string
		dsUser        string
		mid, igDid    string
		user, pass    string
		totp          string
		cookieMode    bool
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in (password, or --cookies if Meta blocks password)",
		Long: `Login options:

  threadterm login              # username + password
  threadterm login --cookies    # guided browser cookies (most reliable)

If password login fails with "no Bearer token" / unexpected error,
Meta blocked the device — use --cookies (30 seconds).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if cookieMode {
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

			wantInteractive := user == "" && pass == "" && cookieString == "" && sessionID == "" && token == ""
			if wantInteractive {
				fmt.Println("threadterm login")
				fmt.Println("Enter your Threads / Instagram username and password.")
				fmt.Println("(If Meta blocks this, we'll switch to easy cookie login.)")
				fmt.Println()
				user, err = promptLine("username: ")
				if err != nil {
					return err
				}
				pass, err = promptPassword("password: ")
				if err != nil {
					return err
				}
				t, _ := promptLine("2FA secret (optional, Enter to skip): ")
				totp = strings.TrimSpace(t)
			}

			if user != "" {
				if pass == "" {
					var err error
					pass, err = promptPassword("password: ")
					if err != nil {
						return err
					}
				}
				fmt.Println("logging in…")
				if err := auth.LoginPasswordTOTP(cfg, user, pass, totp); err != nil {
					fmt.Println()
					fmt.Println(auth.ExplainLoginFailure(err))
					fmt.Println()
					fmt.Print("Switch to cookie login now? [Y/n] ")
					ans, _ := promptLine("")
					ans = strings.ToLower(strings.TrimSpace(ans))
					if ans == "" || ans == "y" || ans == "yes" {
						return auth.GuidedCookieLogin(cfg)
					}
					return auth.ExplainLoginFailure(err)
				}
				fmt.Printf("logged in as @%s · mode=%s\n", cfg.Username, cfg.Mode())
				fmt.Println("try:  threadterm          # TUI")
				fmt.Println("      threadterm feed")
				fmt.Println("      threadterm post \"hello\"")
				return nil
			}

			return fmt.Errorf("nothing to do — run: threadterm login   or   threadterm login --cookies")
		},
	}
	cmd.Flags().BoolVar(&cookieMode, "cookies", false, "guided browser-cookie login (most reliable)")
	cmd.Flags().StringVar(&cookieString, "cookie-string", "", "optional: raw Cookie header from threads.com")
	cmd.Flags().StringVar(&sessionID, "session-id", "", "optional sessionid cookie")
	cmd.Flags().StringVar(&csrf, "csrf", "", "optional csrftoken cookie")
	cmd.Flags().StringVar(&dsUser, "ds-user-id", "", "optional ds_user_id cookie")
	cmd.Flags().StringVar(&mid, "mid", "", "optional mid cookie")
	cmd.Flags().StringVar(&igDid, "ig-did", "", "optional ig_did cookie")
	cmd.Flags().StringVar(&user, "user", "", "Threads/IG username")
	cmd.Flags().StringVar(&pass, "password", "", "password (omit to get a hidden prompt)")
	cmd.Flags().StringVar(&totp, "totp", "", "authenticator 2FA secret (optional)")
	cmd.Flags().StringVar(&token, "token", "", "official Graph access token (optional)")
	cmd.Flags().StringVar(&userID, "user-id", "", "official Graph user id")
	cmd.Flags().IntVar(&port, "port", 8765, "localhost OAuth callback port")
	_ = port
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
			fmt.Println("logged out Â· demo mode")
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
				fmt.Println("available: jade Â· ocean Â· ember Â· mono Â· orchid")
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
	return s[:4] + "â€¦" + s[len(s)-4:]
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
