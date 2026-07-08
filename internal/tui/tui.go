package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/arifaqyl/threadterm/internal/api"
	"github.com/arifaqyl/threadterm/internal/auth"
	"github.com/arifaqyl/threadterm/internal/config"
	"github.com/arifaqyl/threadterm/internal/models"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type view int

const (
	viewWelcome view = iota
	viewFeed
	viewThread
	viewCompose
	viewProfile
	viewHelp
	viewLogin
	viewTheme
)

type feedLoadedMsg struct {
	page models.FeedPage
	err  error
}
type threadLoadedMsg struct {
	th  models.Thread
	err error
}
type profileLoadedMsg struct {
	user  models.User
	posts []models.Post
	err   error
}
type publishedMsg struct {
	res models.PublishResult
	err error
}
type loginDoneMsg struct {
	cfg *config.Config
	err error
}
type statusMsg string

type Model struct {
	cfg    *config.Config
	client api.Client
	width  int
	height int
	styles styles
	theme  Theme

	view   view
	status string
	err    string

	posts   []models.Post
	cursor  int
	thread  *models.Thread
	profile *models.User
	pPosts  []models.Post

	compose textarea.Model
	replyTo string

	// login form
	// 0=menu 1=cookies 2=user 3=pass 4=official-token 5=official-uid 6=oauth
	loginStep  int
	cookieInput textinput.Model
	userInput   textinput.Model
	passInput   textinput.Model
	tokenInput  textinput.Model
	uidInput    textinput.Model

	themeCursor int

	vp    viewport.Model
	ready bool
}

func New(cfg *config.Config, client api.Client) Model {
	th := themeByName(cfg.Theme)
	ta := textarea.New()
	ta.Placeholder = "What's on your mind?"
	ta.CharLimit = 500
	ta.SetHeight(5)
	ta.SetWidth(56)
	ta.ShowLineNumbers = false

	ci := textinput.New()
	ci.Placeholder = "sessionid=…; csrftoken=…; ds_user_id=…; mid=…; ig_did=…"
	ci.CharLimit = 4096
	ci.Width = 56

	ui := textinput.New()
	ui.Placeholder = "username (no @)"
	ui.CharLimit = 64
	ui.Width = 40

	pi := textinput.New()
	pi.Placeholder = "password"
	pi.CharLimit = 128
	pi.Width = 40
	pi.EchoMode = textinput.EchoPassword
	pi.EchoCharacter = '•'

	ti := textinput.New()
	ti.Placeholder = "official Graph access token (optional)"
	ti.CharLimit = 512
	ti.Width = 48
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'

	oid := textinput.New()
	oid.Placeholder = "official user id"
	oid.CharLimit = 64
	oid.Width = 48

	start := viewFeed
	if !cfg.SeenWelcome {
		start = viewWelcome
	}

	return Model{
		cfg:         cfg,
		client:      client,
		theme:       th,
		styles:      makeStyles(th),
		view:        start,
		compose:     ta,
		cookieInput: ci,
		userInput:   ui,
		passInput:   pi,
		tokenInput:  ti,
		uidInput:    oid,
		status:      "ready",
		vp:          viewport.New(80, 20),
	}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tea.SetWindowTitle("threadterm")}
	if m.view != viewWelcome {
		cmds = append(cmds, loadFeed(m.client))
	}
	return tea.Batch(cmds...)
}

func loadFeed(c api.Client) tea.Cmd {
	return func() tea.Msg {
		page, err := c.Feed("", 40)
		return feedLoadedMsg{page: page, err: err}
	}
}

func loadThread(c api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		th, err := c.Thread(id)
		return threadLoadedMsg{th: th, err: err}
	}
}

func loadProfile(c api.Client, username string) tea.Cmd {
	return func() tea.Msg {
		u, posts, err := c.Profile(username)
		return profileLoadedMsg{user: u, posts: posts, err: err}
	}
}

func publish(c api.Client, text, replyTo string) tea.Cmd {
	return func() tea.Msg {
		var res models.PublishResult
		var err error
		if replyTo != "" {
			res, err = c.Reply(replyTo, text)
		} else {
			res, err = c.Publish(text)
		}
		return publishedMsg{res: res, err: err}
	}
}

func likePost(c api.Client, id string, unlike bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if unlike {
			err = c.Unlike(id)
		} else {
			err = c.Like(id)
		}
		if err != nil {
			return statusMsg("like failed: " + err.Error())
		}
		if unlike {
			return statusMsg("unliked")
		}
		return statusMsg("liked ♥")
	}
}

func runOAuth(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		out, err := auth.LoginLocalhost(cfg, 8765)
		return loginDoneMsg{cfg: out, err: err}
	}
}

func saveToken(cfg *config.Config, token, uid string) tea.Cmd {
	return func() tea.Msg {
		if err := auth.SetToken(cfg, token, uid); err != nil {
			return loginDoneMsg{err: err}
		}
		return loginDoneMsg{cfg: cfg}
	}
}

func saveCookies(cfg *config.Config, raw string) tea.Cmd {
	return func() tea.Msg {
		if err := auth.SetSessionFromPaste(cfg, raw); err != nil {
			return loginDoneMsg{err: err}
		}
		return loginDoneMsg{cfg: cfg}
	}
}

func savePassword(cfg *config.Config, user, pass string) tea.Cmd {
	return func() tea.Msg {
		if err := auth.LoginPassword(cfg, user, pass); err != nil {
			return loginDoneMsg{err: err}
		}
		return loginDoneMsg{cfg: cfg}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		m.ready = true
		m.refreshViewport()
		return m, nil

	case feedLoadedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "error"
			return m, nil
		}
		m.posts = msg.page.Posts
		if m.cursor >= len(m.posts) {
			m.cursor = max(0, len(m.posts)-1)
		}
		m.err = ""
		auth := m.client.AuthStatus()
		m.status = fmt.Sprintf("%d posts · %s · %s", len(m.posts), auth.Mode, time.Now().Format("15:04"))
		m.refreshViewport()
		return m, nil

	case threadLoadedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "error"
			return m, nil
		}
		m.thread = &msg.th
		m.view = viewThread
		m.status = fmt.Sprintf("thread · %d replies", len(msg.th.Replies))
		m.refreshViewport()
		return m, nil

	case profileLoadedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "error"
			return m, nil
		}
		m.profile = &msg.user
		m.pPosts = msg.posts
		m.view = viewProfile
		m.status = "@" + msg.user.Username
		m.refreshViewport()
		return m, nil

	case publishedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "publish failed"
			return m, nil
		}
		m.compose.Reset()
		m.replyTo = ""
		m.view = viewFeed
		m.status = "posted · " + msg.res.ID
		return m, loadFeed(m.client)

	case loginDoneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "login failed"
			m.loginStep = 0
			return m, nil
		}
		m.cfg = msg.cfg
		m.client = api.New(m.cfg)
		m.view = viewFeed
		m.loginStep = 0
		m.status = fmt.Sprintf("logged in · @%s", m.cfg.Username)
		m.err = ""
		return m, loadFeed(m.client)

	case statusMsg:
		m.status = string(msg)
		return m, nil

	case tea.KeyMsg:
		switch m.view {
		case viewWelcome:
			return m.updateWelcome(msg)
		case viewCompose:
			return m.updateCompose(msg)
		case viewLogin:
			return m.updateLogin(msg)
		case viewTheme:
			return m.updateTheme(msg)
		case viewHelp:
			if msg.String() == "q" || msg.String() == "esc" || msg.String() == "?" || msg.String() == "h" {
				m.view = viewFeed
				m.refreshViewport()
				return m, nil
			}
			return m, nil
		default:
			return m.updateNav(msg)
		}
	}

	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m Model) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", " ", "1":
		m.cfg.SeenWelcome = true
		_ = m.cfg.Save()
		m.view = viewFeed
		m.status = "loading feed…"
		m.refreshViewport()
		return m, loadFeed(m.client)
	case "2", "l":
		m.cfg.SeenWelcome = true
		_ = m.cfg.Save()
		m.view = viewLogin
		m.loginStep = 0
		m.refreshViewport()
		return m, nil
	case "3", "t":
		m.view = viewTheme
		m.refreshViewport()
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) updateLogin(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.loginStep {
	case 0: // menu
		switch msg.String() {
		case "1", "c":
			m.loginStep = 1
			m.cookieInput.SetValue("")
			m.cookieInput.Focus()
			m.err = ""
			return m, textinput.Blink
		case "2", "w":
			m.loginStep = 2
			m.userInput.SetValue(m.cfg.Username)
			m.userInput.Focus()
			m.err = ""
			return m, textinput.Blink
		case "3", "o":
			m.loginStep = 4
			m.tokenInput.SetValue("")
			m.tokenInput.Focus()
			m.err = ""
			return m, textinput.Blink
		case "4", "d":
			m.cfg.Demo = true
			_ = auth.ClearSession(m.cfg)
			m.cfg.Demo = true
			_ = m.cfg.Save()
			m.client = api.New(m.cfg)
			m.view = viewFeed
			m.status = "demo mode"
			return m, loadFeed(m.client)
		case "esc", "q", "h":
			m.view = viewFeed
			m.refreshViewport()
			return m, nil
		}
	case 1: // cookies
		switch msg.String() {
		case "esc":
			m.loginStep = 0
			m.cookieInput.Blur()
			return m, nil
		case "enter":
			raw := strings.TrimSpace(m.cookieInput.Value())
			if raw == "" {
				m.err = "paste cookies from threads.com DevTools"
				return m, nil
			}
			m.cookieInput.Blur()
			m.status = "saving session…"
			return m, saveCookies(m.cfg, raw)
		}
		var cmd tea.Cmd
		m.cookieInput, cmd = m.cookieInput.Update(msg)
		return m, cmd
	case 2: // username for write
		switch msg.String() {
		case "esc":
			m.loginStep = 0
			m.userInput.Blur()
			return m, nil
		case "enter":
			m.loginStep = 3
			m.userInput.Blur()
			m.passInput.SetValue("")
			m.passInput.Focus()
			return m, textinput.Blink
		}
		var cmd tea.Cmd
		m.userInput, cmd = m.userInput.Update(msg)
		return m, cmd
	case 3: // password
		switch msg.String() {
		case "esc":
			m.loginStep = 2
			m.passInput.Blur()
			m.userInput.Focus()
			return m, textinput.Blink
		case "enter":
			u := strings.TrimSpace(m.userInput.Value())
			p := m.passInput.Value()
			if u == "" || p == "" {
				m.err = "username and password required"
				return m, nil
			}
			m.passInput.Blur()
			m.status = "bloks login…"
			return m, savePassword(m.cfg, u, p)
		}
		var cmd tea.Cmd
		m.passInput, cmd = m.passInput.Update(msg)
		return m, cmd
	case 4: // official token
		switch msg.String() {
		case "esc":
			m.loginStep = 0
			m.tokenInput.Blur()
			return m, nil
		case "enter":
			m.loginStep = 5
			m.tokenInput.Blur()
			m.uidInput.SetValue(m.cfg.UserID)
			m.uidInput.Focus()
			return m, textinput.Blink
		}
		var cmd tea.Cmd
		m.tokenInput, cmd = m.tokenInput.Update(msg)
		return m, cmd
	case 5: // official uid
		switch msg.String() {
		case "esc":
			m.loginStep = 4
			m.uidInput.Blur()
			m.tokenInput.Focus()
			return m, textinput.Blink
		case "enter":
			tok := strings.TrimSpace(m.tokenInput.Value())
			uid := strings.TrimSpace(m.uidInput.Value())
			if tok == "" || uid == "" {
				m.err = "token and user id required"
				return m, nil
			}
			m.uidInput.Blur()
			m.status = "saving…"
			return m, saveToken(m.cfg, tok, uid)
		}
		var cmd tea.Cmd
		m.uidInput, cmd = m.uidInput.Update(msg)
		return m, cmd
	case 6:
		if msg.String() == "esc" || msg.String() == "q" {
			m.loginStep = 0
			m.status = "oauth cancelled"
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateTheme(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		m.themeCursor = (m.themeCursor + 1) % len(themes)
		return m, nil
	case "k", "up":
		m.themeCursor = (m.themeCursor - 1 + len(themes)) % len(themes)
		return m, nil
	case "enter", " ":
		th := themes[m.themeCursor]
		m.applyTheme(th)
		m.cfg.Theme = th.Name
		_ = m.cfg.Save()
		m.view = viewFeed
		m.status = "theme · " + th.Name
		m.refreshViewport()
		return m, nil
	case "esc", "q", "h":
		m.view = viewFeed
		m.refreshViewport()
		return m, nil
	}
	return m, nil
}

func (m *Model) applyTheme(th Theme) {
	m.theme = th
	m.styles = makeStyles(th)
}

func (m Model) updateCompose(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = viewFeed
		m.compose.Blur()
		m.replyTo = ""
		m.refreshViewport()
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+s":
		text := strings.TrimSpace(m.compose.Value())
		if text == "" {
			m.status = "empty post"
			return m, nil
		}
		m.status = "publishing…"
		m.compose.Blur()
		return m, publish(m.client, text, m.replyTo)
	}
	var cmd tea.Cmd
	m.compose, cmd = m.compose.Update(msg)
	return m, cmd
}

func (m Model) updateNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		if m.view != viewFeed {
			m.view = viewFeed
			m.thread = nil
			m.profile = nil
			m.refreshViewport()
			return m, nil
		}
		return m, tea.Quit
	case "?":
		m.view = viewHelp
		m.refreshViewport()
		return m, nil
	case "a", "A":
		m.view = viewLogin
		m.loginStep = 0
		m.err = ""
		m.refreshViewport()
		return m, nil
	case "t":
		// sync cursor to current theme
		for i, th := range themes {
			if th.Name == m.theme.Name {
				m.themeCursor = i
				break
			}
		}
		m.view = viewTheme
		m.refreshViewport()
		return m, nil
	case "T":
		th := nextTheme(m.theme.Name)
		m.applyTheme(th)
		m.cfg.Theme = th.Name
		_ = m.cfg.Save()
		m.status = "theme · " + th.Name
		m.refreshViewport()
		return m, nil
	case "r":
		m.status = "refreshing…"
		return m, loadFeed(m.client)
	case "j", "down":
		if m.cursor < len(m.posts)-1 {
			m.cursor++
			m.refreshViewport()
		}
		return m, nil
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.refreshViewport()
		}
		return m, nil
	case "g":
		m.cursor = 0
		m.refreshViewport()
		return m, nil
	case "G":
		if len(m.posts) > 0 {
			m.cursor = len(m.posts) - 1
			m.refreshViewport()
		}
		return m, nil
	case "enter", "l":
		if m.view == viewFeed && len(m.posts) > 0 {
			m.status = "loading thread…"
			return m, loadThread(m.client, m.posts[m.cursor].ID)
		}
		return m, nil
	case "c", "n":
		m.view = viewCompose
		m.replyTo = ""
		m.compose.Placeholder = "What's on your mind?"
		m.compose.Focus()
		return m, textarea.Blink
	case "R":
		if len(m.posts) > 0 {
			m.view = viewCompose
			m.replyTo = m.posts[m.cursor].ID
			m.compose.Placeholder = "Reply to @" + m.posts[m.cursor].Username
			m.compose.Focus()
			return m, textarea.Blink
		}
		return m, nil
	case "L":
		if len(m.posts) > 0 {
			p := &m.posts[m.cursor]
			return m, likePost(m.client, p.ID, p.LikedByMe)
		}
		return m, nil
	case "p":
		if len(m.posts) > 0 {
			m.status = "loading profile…"
			return m, loadProfile(m.client, m.posts[m.cursor].Username)
		}
		return m, nil
	case "esc", "backspace", "h":
		if m.view != viewFeed {
			m.view = viewFeed
			m.thread = nil
			m.profile = nil
			m.refreshViewport()
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m *Model) layout() {
	sideW := 22
	if m.width < 70 {
		sideW = 0
	}
	header := 3
	footer := 2
	m.vp.Width = max(20, m.width-sideW-2)
	m.vp.Height = max(1, m.height-header-footer)
	m.compose.SetWidth(min(64, m.vp.Width-4))
}

func (m *Model) refreshViewport() {
	switch m.view {
	case viewFeed:
		m.vp.SetContent(m.renderFeedBody())
	case viewThread:
		m.vp.SetContent(m.renderThreadBody())
	case viewProfile:
		m.vp.SetContent(m.renderProfileBody())
	case viewHelp:
		m.vp.SetContent(m.renderHelp())
	case viewWelcome, viewLogin, viewTheme, viewCompose:
		// rendered in View()
	}
}

func (m Model) View() string {
	if !m.ready {
		return "\n  threadterm · starting…"
	}

	switch m.view {
	case viewWelcome:
		return m.renderWelcome()
	case viewLogin:
		return m.frame(m.renderLogin())
	case viewTheme:
		return m.frame(m.renderThemePicker())
	case viewCompose:
		return m.frame(m.renderCompose())
	default:
		body := m.vp.View()
		if m.width >= 70 {
			body = lipgloss.JoinHorizontal(lipgloss.Top, m.renderSidebar(), body)
		}
		return m.frame(body)
	}
}

func (m Model) frame(body string) string {
	auth := m.client.AuthStatus()
	title := m.styles.brand.Render(" threadterm ")
	modeLabel := auth.Mode
	if auth.Username != "" && auth.Mode != "demo" {
		modeLabel = auth.Mode + " · @" + auth.Username
	} else if auth.Mode == "demo" {
		modeLabel = "demo"
	}
	mode := m.styles.mode.Render(" " + modeLabel + " ")
	themeBadge := m.styles.muted.Render("  " + m.theme.Name)
	header := lipgloss.JoinHorizontal(lipgloss.Center, title, " ", mode, themeBadge)

	footerBits := []string{m.status}
	if m.err != "" {
		footerBits = append(footerBits, m.styles.err.Render(m.err))
	}
	footerBits = append(footerBits, m.styles.muted.Render("? help · a login · t theme · q quit"))
	footer := m.styles.footer.Render("  " + strings.Join(footerBits, "  ·  "))

	return lipgloss.JoinVertical(lipgloss.Left, header, "", body, footer)
}

func (m Model) renderSidebar() string {
	s := m.styles
	auth := m.client.AuthStatus()
	items := []string{
		s.accent.Render("NAV"),
		m.navItem("f", "feed", m.view == viewFeed),
		m.navItem("c", "compose", m.view == viewCompose),
		m.navItem("a", "login", m.view == viewLogin),
		m.navItem("t", "theme", m.view == viewTheme),
		m.navItem("?", "help", m.view == viewHelp),
		"",
		s.accent.Render("STATUS"),
		s.muted.Render("mode  "+auth.Mode),
		s.muted.Render("theme "+m.theme.Name),
	}
	if auth.Username != "" {
		items = append(items, s.muted.Render("@"+auth.Username))
	}
	if m.view == viewFeed && len(m.posts) > 0 {
		items = append(items, "", s.accent.Render("SELECTED"))
		p := m.posts[m.cursor]
		items = append(items, s.muted.Render("@"+p.Username))
		items = append(items, s.muted.Render(fmt.Sprintf("♥ %s", formatCount(p.LikeCount))))
	}
	inner := strings.Join(items, "\n")
	return s.sidebar.Width(20).Height(m.vp.Height).Render(inner)
}

func (m Model) navItem(key, label string, active bool) string {
	if active {
		return m.styles.accent.Render("▸ " + key + "  " + label)
	}
	return m.styles.muted.Render("  " + key + "  " + label)
}

func (m Model) renderWelcome() string {
	s := m.styles
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.theme.Accent)).
		Padding(1, 3).
		Width(min(64, m.width-4))

	body := strings.Join([]string{
		s.brand.Render(" threadterm "),
		"",
		s.title.Render("Threads in your terminal"),
		s.muted.Render("TUI + CLI · demo offline · official API when live"),
		"",
		s.accent.Render("How to use"),
		"  j/k     move through posts",
		"  enter   open a thread",
		"  c       compose a post",
		"  R       reply · L like · p profile",
		"  a       login (token or OAuth)",
		"  t       pick a color theme",
		"  ?       full help · q quit",
		"",
		s.accent.Render("Get started"),
		"  1  enter   browse demo feed",
		"  2  l       login to your Threads",
		"  3  t       choose a theme first",
		"",
		s.hint.Render("Live mode = paste threads.com cookies. No developer account."),
	}, "\n")

	content := box.Render(body)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m Model) renderLogin() string {
	s := m.styles
	var body string
	switch m.loginStep {
	case 0:
		body = strings.Join([]string{
			s.section.Render(" LOGIN "),
			"",
			"No Meta developer app needed.",
			"Same idea as Twitter/X CLIs: use your browser session.",
			"",
			s.accent.Render("1 / c") + "  Paste cookies from threads.com  ← recommended",
			s.accent.Render("2 / w") + "  Write login (username + password)",
			s.accent.Render("3 / o") + "  Official Graph token (optional)",
			s.accent.Render("4 / d") + "  Stay in demo mode",
			"",
			s.muted.Render("Cookies = read feeds/profiles"),
			s.muted.Render("Write login = post / like / reply"),
			s.muted.Render("docs/AUTH.md  ·  esc back"),
		}, "\n")
	case 1:
		body = strings.Join([]string{
			s.section.Render(" COOKIES "),
			"",
			"1. Open https://www.threads.com (logged in)",
			"2. DevTools → Application → Cookies",
			"3. Paste sessionid + csrftoken + ds_user_id (+ mid, ig_did)",
			"",
			m.cookieInput.View(),
			"",
			s.muted.Render("enter save · esc back"),
		}, "\n")
	case 2:
		body = strings.Join([]string{
			s.section.Render(" WRITE LOGIN "),
			"",
			"Username (Instagram / Threads):",
			"",
			m.userInput.View(),
			"",
			s.muted.Render("enter next · esc back"),
		}, "\n")
	case 3:
		body = strings.Join([]string{
			s.section.Render(" PASSWORD "),
			"",
			"Password (sent to Instagram Bloks login — not Meta Graph):",
			"",
			m.passInput.View(),
			"",
			s.muted.Render("enter login · esc back"),
			s.hint.Render("May hit checkpoint 2FA — use cookies-only if so."),
		}, "\n")
	case 4:
		body = strings.Join([]string{
			s.section.Render(" OFFICIAL TOKEN "),
			"",
			"Optional Graph API access token:",
			"",
			m.tokenInput.View(),
			"",
			s.muted.Render("enter next · esc back"),
		}, "\n")
	case 5:
		body = strings.Join([]string{
			s.section.Render(" USER ID "),
			"",
			"Official Threads user id:",
			"",
			m.uidInput.View(),
			"",
			s.muted.Render("enter save · esc back"),
		}, "\n")
	case 6:
		body = strings.Join([]string{
			s.section.Render(" OAUTH "),
			"",
			"Waiting on http://127.0.0.1:8765/callback …",
			s.muted.Render("Prefer cookie login unless you have a Meta app."),
		}, "\n")
	}
	return s.compose.Render(body)
}

func (m Model) renderThemePicker() string {
	s := m.styles
	var b strings.Builder
	b.WriteString(s.section.Render(" THEMES "))
	b.WriteString("\n\n")
	b.WriteString(s.muted.Render("Personalize colors. Enter to apply · T cycles anytime.\n\n"))
	for i, th := range themes {
		swatch := lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.BrandFg)).
			Background(lipgloss.Color(th.Accent)).
			Render(" ██ ")
		line := fmt.Sprintf("%s  %s", swatch, th.Name)
		if i == m.themeCursor {
			line = s.accent.Render("▸ ") + line + s.muted.Render("  ←")
		} else {
			line = "  " + line
		}
		if th.Name == m.theme.Name {
			line += s.tag.Render("  active")
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + s.muted.Render("j/k move · enter apply · esc back"))
	return b.String()
}

func (m Model) renderCompose() string {
	s := m.styles
	label := "COMPOSE"
	if m.replyTo != "" {
		label = "REPLY"
	}
	return s.compose.Render(
		s.section.Render(" "+label+" ") + "\n\n" +
			m.compose.View() + "\n\n" +
			s.muted.Render(fmt.Sprintf("ctrl+s post · esc cancel · %d/500", len([]rune(m.compose.Value())))),
	)
}

func (m Model) renderFeedBody() string {
	s := m.styles
	if len(m.posts) == 0 {
		return s.muted.Render("\n  no posts — press r to refresh, c to compose, a to login\n")
	}
	var b strings.Builder
	b.WriteString(s.muted.Render(fmt.Sprintf("  FEED  ·  %d  ·  j/k move  ·  enter open\n\n", len(m.posts))))
	for i, p := range m.posts {
		b.WriteString(m.renderPostCard(p, i == m.cursor))
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) renderThreadBody() string {
	s := m.styles
	if m.thread == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(s.section.Render(" THREAD "))
	b.WriteString("\n\n")
	b.WriteString(m.renderPostCard(m.thread.Root, true))
	b.WriteString("\n")
	b.WriteString(s.section.Render(fmt.Sprintf(" REPLIES · %d ", len(m.thread.Replies))))
	b.WriteString("\n\n")
	for _, r := range m.thread.Replies {
		b.WriteString(m.renderPostCard(r, false))
		b.WriteString("\n")
	}
	b.WriteString(s.muted.Render("\n  h/esc back · R reply · L like\n"))
	return b.String()
}

func (m Model) renderProfileBody() string {
	s := m.styles
	if m.profile == nil {
		return ""
	}
	u := m.profile
	var b strings.Builder
	b.WriteString(s.brand.Render(" @" + u.Username + " "))
	if u.Verified {
		b.WriteString(s.accent.Render(" ✓"))
	}
	b.WriteString("\n")
	if u.Name != "" {
		b.WriteString(s.title.Render(u.Name) + "\n")
	}
	if u.Bio != "" {
		b.WriteString(s.muted.Render(u.Bio) + "\n")
	}
	b.WriteString(s.muted.Render(fmt.Sprintf("%s followers", formatCount(u.Followers))) + "\n\n")
	for _, p := range m.pPosts {
		b.WriteString(m.renderPostCard(p, false))
		b.WriteString("\n")
	}
	b.WriteString(s.muted.Render("\n  h/esc back\n"))
	return b.String()
}

func (m Model) renderHelp() string {
	s := m.styles
	return strings.Join([]string{
		s.section.Render(" HELP "),
		"",
		s.accent.Render("Navigation"),
		"  j/k  ↓/↑      move",
		"  enter / l     open thread",
		"  g / G         top / bottom",
		"  h / esc       back",
		"  r             refresh feed",
		"",
		s.accent.Render("Actions"),
		"  c / n         compose",
		"  R             reply to selected",
		"  L             like / unlike",
		"  p             author profile",
		"",
		s.accent.Render("Account & look"),
		"  a             login / demo / OAuth",
		"  t             theme picker",
		"  T             cycle theme",
		"  ?             this help",
		"  q             quit (or back)",
		"",
		s.accent.Render("CLI (outside TUI)"),
		"  threadterm feed --json",
		"  threadterm post \"hello\"",
		"  threadterm login",
		"  threadterm search golang",
		"  threadterm doctor",
		"",
		s.muted.Render("Auth guide: docs/AUTH.md · Launch copy: docs/LAUNCH_X.md"),
		s.hint.Render("esc / ? to close"),
	}, "\n")
}

func (m Model) renderPostCard(p models.Post, selected bool) string {
	s := m.styles
	w := max(36, m.vp.Width-4)
	handle := s.accent.Render("@" + p.Username)
	ago := s.muted.Render(relTime(p.Timestamp))
	meta := fmt.Sprintf("%s  %s", handle, ago)
	if p.TopicTag != "" {
		meta += "  " + s.tag.Render("#"+p.TopicTag)
	}
	text := wordWrap(p.Text, w-4)
	stats := s.muted.Render(fmt.Sprintf("♥ %s   💬 %s   ↻ %s", formatCount(p.LikeCount), formatCount(p.ReplyCount), formatCount(p.RepostCount)))
	if p.LikedByMe {
		stats = s.accent.Render(fmt.Sprintf("♥ %s", formatCount(p.LikeCount))) + s.muted.Render(fmt.Sprintf("   💬 %s   ↻ %s", formatCount(p.ReplyCount), formatCount(p.RepostCount)))
	}
	inner := meta + "\n" + text + "\n" + stats
	if selected {
		return s.selected.Width(w).Render(inner)
	}
	return s.card.Width(w).Render(inner)
}

func relTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func formatCount(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func wordWrap(s string, width int) string {
	if width < 20 {
		width = 20
	}
	if strings.Contains(s, "\n") {
		return strings.TrimSpace(s)
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return s
	}
	var lines []string
	var line string
	for _, w := range words {
		if line == "" {
			line = w
			continue
		}
		if len(line)+1+len(w) > width {
			lines = append(lines, line)
			line = w
		} else {
			line += " " + w
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Run starts the Bubble Tea program.
func Run(cfg *config.Config, client api.Client) error {
	p := tea.NewProgram(New(cfg, client), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
