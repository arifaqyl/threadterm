package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/arifaqyl/threadterm/internal/api"
	"github.com/arifaqyl/threadterm/internal/models"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type view int

const (
	viewFeed view = iota
	viewThread
	viewCompose
	viewProfile
	viewHelp
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
type statusMsg string

type Model struct {
	client api.Client
	width  int
	height int

	view   view
	status string
	err    string

	posts    []models.Post
	cursor   int
	thread   *models.Thread
	profile  *models.User
	pPosts   []models.Post

	compose textarea.Model
	replyTo string

	vp viewport.Model
	ready bool
}

func New(client api.Client) Model {
	ta := textarea.New()
	ta.Placeholder = "What's new?"
	ta.CharLimit = 500
	ta.SetHeight(6)
	ta.SetWidth(60)
	ta.ShowLineNumbers = false
	return Model{
		client:  client,
		view:    viewFeed,
		compose: ta,
		status:  "loading feed…",
		vp:      viewport.New(80, 20),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadFeed(m.client), tea.SetWindowTitle("threadterm"))
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		header := 4
		footer := 2
		m.vp.Width = msg.Width
		m.vp.Height = max(1, msg.Height-header-footer)
		m.compose.SetWidth(min(72, msg.Width-4))
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
		m.cursor = 0
		m.err = ""
		auth := m.client.AuthStatus()
		m.status = fmt.Sprintf("%s · %d posts · %s", auth.Mode, len(m.posts), time.Now().Format("15:04"))
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

	case statusMsg:
		m.status = string(msg)
		return m, nil

	case tea.KeyMsg:
		if m.view == viewCompose {
			return m.updateCompose(msg)
		}
		switch msg.String() {
		case "ctrl+c", "q":
			if m.view != viewFeed && m.view != viewHelp {
				m.view = viewFeed
				m.thread = nil
				m.profile = nil
				m.refreshViewport()
				return m, nil
			}
			if m.view == viewHelp {
				m.view = viewFeed
				m.refreshViewport()
				return m, nil
			}
			return m, tea.Quit
		case "?":
			m.view = viewHelp
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
				id := m.posts[m.cursor].ID
				m.status = "loading thread…"
				return m, loadThread(m.client, id)
			}
			return m, nil
		case "c", "n":
			m.view = viewCompose
			m.replyTo = ""
			m.compose.Placeholder = "What's new?"
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
	}

	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
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
	case "ctrl+s", "ctrl+enter":
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

func (m *Model) refreshViewport() {
	switch m.view {
	case viewFeed:
		m.vp.SetContent(m.renderFeedBody())
	case viewThread:
		m.vp.SetContent(m.renderThreadBody())
	case viewProfile:
		m.vp.SetContent(m.renderProfileBody())
	case viewHelp:
		m.vp.SetContent(helpText)
	case viewCompose:
		// compose rendered in View()
	}
}

func (m Model) View() string {
	if !m.ready {
		return "\n  threadterm · starting…"
	}
	auth := m.client.AuthStatus()
	title := brandStyle.Render(" threadterm ")
	mode := modeStyle.Render(fmt.Sprintf(" %s ", auth.Mode))
	if auth.Username != "" {
		mode = modeStyle.Render(fmt.Sprintf(" %s · @%s ", auth.Mode, auth.Username))
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top, title, " ", mode)
	sub := mutedStyle.Render("  Threads in your terminal  ·  ? help  ·  q quit")

	var body string
	switch m.view {
	case viewCompose:
		label := "compose"
		if m.replyTo != "" {
			label = "reply"
		}
		body = composeBox.Render(
			accentStyle.Render(label)+"\n\n"+
				m.compose.View()+"\n\n"+
				mutedStyle.Render("ctrl+s post · esc cancel · "+fmt.Sprintf("%d/500", len(m.compose.Value()))),
		)
	default:
		body = m.vp.View()
	}

	footerBits := []string{m.status}
	if m.err != "" {
		footerBits = append(footerBits, errStyle.Render(m.err))
	}
	footer := footerStyle.Render("  " + strings.Join(footerBits, "  ·  "))

	return lipgloss.JoinVertical(lipgloss.Left, header, sub, "", body, footer)
}

func (m Model) renderFeedBody() string {
	if len(m.posts) == 0 {
		return mutedStyle.Render("\n  no posts — press r to refresh, or c to compose\n")
	}
	var b strings.Builder
	for i, p := range m.posts {
		selected := i == m.cursor
		b.WriteString(renderPostCard(p, selected, m.width))
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) renderThreadBody() string {
	if m.thread == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(sectionStyle.Render(" THREAD "))
	b.WriteString("\n")
	b.WriteString(renderPostCard(m.thread.Root, true, m.width))
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render(fmt.Sprintf(" REPLIES (%d) ", len(m.thread.Replies))))
	b.WriteString("\n")
	for _, r := range m.thread.Replies {
		b.WriteString(renderPostCard(r, false, m.width))
		b.WriteString("\n")
	}
	b.WriteString(mutedStyle.Render("\n  h/esc back · R reply · L like\n"))
	return b.String()
}

func (m Model) renderProfileBody() string {
	if m.profile == nil {
		return ""
	}
	u := m.profile
	var b strings.Builder
	b.WriteString(brandStyle.Render(" @" + u.Username + " "))
	if u.Verified {
		b.WriteString(accentStyle.Render(" ✓"))
	}
	b.WriteString("\n")
	if u.Name != "" {
		b.WriteString(u.Name + "\n")
	}
	if u.Bio != "" {
		b.WriteString(mutedStyle.Render(u.Bio) + "\n")
	}
	b.WriteString(mutedStyle.Render(fmt.Sprintf("%s followers", formatCount(u.Followers))) + "\n\n")
	for _, p := range m.pPosts {
		b.WriteString(renderPostCard(p, false, m.width))
		b.WriteString("\n")
	}
	return b.String()
}

func renderPostCard(p models.Post, selected bool, width int) string {
	w := max(40, width-4)
	handle := accentStyle.Render("@" + p.Username)
	ago := mutedStyle.Render(relTime(p.Timestamp))
	meta := fmt.Sprintf("%s  %s", handle, ago)
	if p.TopicTag != "" {
		meta += "  " + tagStyle.Render("#"+p.TopicTag)
	}
	text := wordWrap(p.Text, w-4)
	stats := mutedStyle.Render(fmt.Sprintf("♥ %s  💬 %s  ↻ %s", formatCount(p.LikeCount), formatCount(p.ReplyCount), formatCount(p.RepostCount)))
	if p.LikedByMe {
		stats = accentStyle.Render(fmt.Sprintf("♥ %s", formatCount(p.LikeCount))) + mutedStyle.Render(fmt.Sprintf("  💬 %s  ↻ %s", formatCount(p.ReplyCount), formatCount(p.RepostCount)))
	}
	inner := meta + "\n" + text + "\n" + stats
	if selected {
		return selectedCard.Width(w).Render(inner)
	}
	return card.Width(w).Render(inner)
}

var (
	brandStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0A0A0A")).Background(lipgloss.Color("#E8E6E1"))
	modeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#E8E6E1")).Background(lipgloss.Color("#2A5A4A"))
	accentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#3D9B7A")).Bold(true)
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7A7A72"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#C45C4A"))
	tagStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#C4A35A"))
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0A0A0A")).Background(lipgloss.Color("#3D9B7A"))
	footerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#9A9A90"))
	card         = lipgloss.NewStyle().Padding(0, 1).MarginLeft(1).Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("#2A2A28"))
	selectedCard = lipgloss.NewStyle().Padding(0, 1).MarginLeft(1).Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("#3D9B7A")).Background(lipgloss.Color("#141816"))
	composeBox   = lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#3D9B7A"))
)

const helpText = `
  KEYBOARD

  j/k · ↓/↑     move
  enter / l     open thread
  c / n         compose
  R             reply to selected
  L             like / unlike
  p             open author profile
  r             refresh feed
  g / G         top / bottom
  h / esc       back
  ?             this help
  q             quit (or back)

  CLI

  threadterm                  open TUI
  threadterm feed --json
  threadterm post "hello"
  threadterm search "golang"
  threadterm login
  threadterm whoami

  AUTH

  Demo mode works offline (default).
  Live mode: Meta Threads Graph API token.
  See docs/AUTH.md
`

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
	// preserve intentional newlines roughly
	if strings.Contains(s, "\n") {
		return strings.TrimSpace(s)
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
func Run(client api.Client) error {
	p := tea.NewProgram(New(client), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
