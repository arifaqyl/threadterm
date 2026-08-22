package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/arifaqyl/threadterm/internal/api"
	"github.com/arifaqyl/threadterm/internal/auth"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

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
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "error"
			m.refreshViewport()
			return m, nil
		}
		m.posts = msg.page.Posts
		m.feedSource = msg.page.Source
		m.feedHint = msg.page.Hint
		if msg.query != "" {
			m.lastQuery = msg.query
		} else if m.feedSource != "search" && m.feedSource != "search-browser" && m.feedSource != "search-web" {
			m.lastQuery = ""
		}
		m.cursor = 0
		if m.cursor >= len(m.posts) {
			m.cursor = max(0, len(m.posts)-1)
		}
		m.err = ""
		src := m.feedSource
		if src == "" {
			src = "feed"
		}
		if m.lastQuery != "" && (src == "search" || src == "search-browser" || src == "search-web") {
			m.status = fmt.Sprintf("%d posts · %s · query: %q · %s", len(m.posts), src, m.lastQuery, time.Now().Format("15:04"))
		} else {
			m.status = fmt.Sprintf("%d posts · %s · %s", len(m.posts), src, time.Now().Format("15:04"))
		}
		if m.feedHint != "" && len(m.posts) == 0 {
			m.status = m.feedHint
		}
		m.view = viewFeed
		m.refreshViewport()
		m.vp.GotoTop()
		m.ensureCursorVisible()
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
		m.vp.GotoTop()
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
		m.vp.GotoTop()
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
			return m.updateHelp(msg)
		case viewSearch:
			return m.updateSearch(msg)
		default:
			return m.updateNav(msg)
		}
	}

	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.view = viewFeed
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.refreshViewport()
		return m, nil
	case "enter":
		q := strings.TrimSpace(m.searchInput.Value())
		m.searchInput.Blur()
		m.view = viewFeed
		if q == "" {
			m.refreshViewport()
			return m, nil
		}
		m.feedSource = "search"
		m.feedHint = ""
		m.lastQuery = q
		m.status = "searching " + q + "…"
		m.loading = true
		m.refreshViewport()
		return m, loadSearch(m.client, q)
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "?", "h":
		m.view = viewFeed
		m.refreshViewport()
		m.ensureCursorVisible()
		return m, nil
	case "j", "down", "ctrl+d", "pgdown":
		m.vp.LineDown(3)
		return m, nil
	case "k", "up", "ctrl+u", "pgup":
		m.vp.LineUp(3)
		return m, nil
	case "g":
		m.vp.GotoTop()
		return m, nil
	case "G":
		m.vp.GotoBottom()
		return m, nil
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
		case "1", "b":
			m.status = "reading browser cookies…"
			m.err = ""
			return m, func() tea.Msg {
				if err := auth.LoginFromBrowser(m.cfg); err != nil {
					return loginDoneMsg{err: err}
				}
				return loginDoneMsg{cfg: m.cfg}
			}
		case "2", "w":
			m.loginStep = 2
			m.userInput.SetValue(m.cfg.Username)
			m.userInput.Focus()
			m.err = ""
			return m, textinput.Blink
		case "3", "c":
			m.loginStep = 1
			m.cookieInput.SetValue("")
			m.cookieInput.Focus()
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
		m.vp.GotoTop()
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
	case "r", "f":
		m.lastQuery = ""
		m.status = "refreshing following…"
		m.loading = true
		m.refreshViewport()
		return m, loadFeed(m.client)
	case "d":
		m.lastQuery = ""
		m.status = "loading public discover…"
		m.loading = true
		m.refreshViewport()
		return m, loadDiscover(m.client)
	case "/":
		m.view = viewSearch
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		return m, textinput.Blink
	case "y", "Y":
		if len(m.posts) == 0 || m.cursor < 0 || m.cursor >= len(m.posts) {
			return m, nil
		}
		p := m.posts[m.cursor]
		text := p.Text
		label := "copied text"
		if p.Permalink != "" {
			text = p.Permalink
			label = "copied link"
		} else if text == "" {
			text = p.ID
			label = "copied id"
		}
		if err := clipboard.WriteAll(text); err != nil {
			m.status = "copy failed: " + err.Error()
		} else {
			m.status = label
		}
		return m, nil
	case "j", "down":
		if m.view == viewFeed {
			if m.cursor < len(m.posts)-1 {
				m.cursor++
				m.refreshViewport()
				m.ensureCursorVisible()
			}
			return m, nil
		}
		m.vp.LineDown(1)
		return m, nil
	case "k", "up":
		if m.view == viewFeed {
			if m.cursor > 0 {
				m.cursor--
				m.refreshViewport()
				m.ensureCursorVisible()
			}
			return m, nil
		}
		m.vp.LineUp(1)
		return m, nil
	case "pgdown", "ctrl+d":
		if m.view == viewFeed && len(m.posts) > 0 {
			step := max(1, m.vp.Height/4)
			m.cursor = min(len(m.posts)-1, m.cursor+step)
			m.refreshViewport()
			m.ensureCursorVisible()
			return m, nil
		}
		m.vp.HalfViewDown()
		return m, nil
	case "pgup", "ctrl+u":
		if m.view == viewFeed && len(m.posts) > 0 {
			step := max(1, m.vp.Height/4)
			m.cursor = max(0, m.cursor-step)
			m.refreshViewport()
			m.ensureCursorVisible()
			return m, nil
		}
		m.vp.HalfViewUp()
		return m, nil
	case "g":
		if m.view == viewFeed {
			m.cursor = 0
			m.refreshViewport()
			m.vp.GotoTop()
			return m, nil
		}
		m.vp.GotoTop()
		return m, nil
	case "G":
		if m.view == viewFeed && len(m.posts) > 0 {
			m.cursor = len(m.posts) - 1
			m.refreshViewport()
			m.ensureCursorVisible()
			return m, nil
		}
		m.vp.GotoBottom()
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
			m.ensureCursorVisible()
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
	y := m.vp.YOffset
	switch m.view {
	case viewFeed:
		body, offsets := m.renderFeedBody()
		m.feedOffsets = offsets
		m.vp.SetContent(body)
	case viewThread:
		m.vp.SetContent(m.renderThreadBody())
	case viewProfile:
		m.vp.SetContent(m.renderProfileBody())
	case viewHelp:
		m.vp.SetContent(m.renderHelp())
	case viewWelcome, viewLogin, viewTheme, viewCompose:
		return
	}
	// Keep prior scroll unless content is shorter.
	m.vp.SetYOffset(y)
}

// ensureCursorVisible scrolls the feed viewport so the selected post is on screen.
func (m *Model) ensureCursorVisible() {
	if m.view != viewFeed || len(m.feedOffsets) == 0 || m.cursor < 0 || m.cursor >= len(m.feedOffsets) {
		return
	}
	start := m.feedOffsets[m.cursor]
	end := m.vp.TotalLineCount()
	if m.cursor+1 < len(m.feedOffsets) {
		end = m.feedOffsets[m.cursor+1]
	}
	height := max(1, m.vp.Height)
	y := m.vp.YOffset

	// If selected block starts above the view, scroll up.
	if start < y {
		m.vp.SetYOffset(start)
		return
	}
	// If selected block ends below the view, scroll down enough to show it.
	if end > y+height {
		m.vp.SetYOffset(max(0, end-height))
	}
}
