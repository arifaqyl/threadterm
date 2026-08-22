package tui

import (
	"fmt"
	"strings"

	"github.com/arifaqyl/threadterm/internal/models"
	"github.com/charmbracelet/lipgloss"
)

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
	case viewSearch:
		return m.frame(m.renderSearch())
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
	footerBits = append(footerBits, m.styles.muted.Render("/ search · y copy · ? help · q quit"))
	footer := m.styles.footer.Render("  " + strings.Join(footerBits, "  ·  "))

	return lipgloss.JoinVertical(lipgloss.Left, header, "", body, footer)
}

func (m Model) renderSidebar() string {
	s := m.styles
	auth := m.client.AuthStatus()
	items := []string{
		s.accent.Render("NAV"),
		m.navItem("f", "feed", m.view == viewFeed && m.feedSource != "search" && m.feedSource != "discover"),
		m.navItem("/", "search", m.view == viewSearch || m.feedSource == "search"),
		m.navItem("d", "discover", m.feedSource == "discover"),
		m.navItem("c", "compose", m.view == viewCompose),
		m.navItem("a", "login", m.view == viewLogin),
		m.navItem("t", "theme", m.view == viewTheme),
		m.navItem("?", "help", m.view == viewHelp),
		"",
		s.accent.Render("STATUS"),
		s.muted.Render("mode  " + auth.Mode),
		s.muted.Render("theme " + m.theme.Name),
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

func (m Model) renderFeedBody() (string, []int) {
	s := m.styles
	if m.loading && len(m.posts) == 0 {
		switch m.feedSource {
		case "search":
			return s.muted.Render("\n  searching posts...\n"), nil
		case "discover":
			return s.muted.Render("\n  loading discover...\n"), nil
		default:
			return s.muted.Render("\n  loading your following feed...\n  (first load can take a few seconds)\n"), nil
		}
	}
	if len(m.posts) == 0 {
		hint := m.feedHint
		if hint == "" {
			switch m.feedSource {
			case "search":
				hint = "no posts found for that query"
			case "discover":
				hint = "discover is empty right now"
			default:
				hint = "no posts from people you follow"
			}
		}
		return s.muted.Render(fmt.Sprintf(
			"\n  %s\n\n  /  search posts\n  d  public discover (not your feed)\n  r  refresh following\n",
			hint,
		)), nil
	}
	src := m.feedSource
	if src == "" {
		src = "feed"
	}
	var b strings.Builder
	header := s.muted.Render(fmt.Sprintf("  %s  ·  %d  ·  j/k  ·  / search  ·  y copy  ·  enter open\n\n", strings.ToUpper(src), len(m.posts)))
	if m.lastQuery != "" && (src == "search" || src == "search-browser" || src == "search-web") {
		header = s.accent.Render("  SEARCH MODE  ") + " " +
			s.muted.Render(fmt.Sprintf("query: %q  ·  %d results  ·  / new search  ·  enter open\n\n", m.lastQuery, len(m.posts)))
	}
	b.WriteString(header)
	offsets := make([]int, len(m.posts))
	line := strings.Count(header, "\n")
	for i, p := range m.posts {
		offsets[i] = line
		card := m.renderPostCard(p, i == m.cursor) + "\n"
		b.WriteString(card)
		line += strings.Count(card, "\n")
	}
	return b.String(), offsets
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
		"  j/k  ↓/↑      move (follows selection)",
		"  pgup/pgdn     jump",
		"  /             search posts",
		"  y             copy link/text (clipboard)",
		"  d             public discover (not your feed)",
		"  f / r         your following feed",
		"  enter / l     open thread",
		"  g / G         top / bottom",
		"  h / esc       back",
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
		"  threadterm feed --discover",
		"  threadterm search malaysia --json",
		"  threadterm post \"hello\"",
		"  threadterm login",
		"",
		s.muted.Render("Select/copy text: mouse select works in terminal (mouse capture off)."),
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
