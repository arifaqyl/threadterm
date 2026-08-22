package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
		s.muted.Render("TUI + CLI · demo offline · browser-cookie login"),
		"",
		s.accent.Render("How to use"),
		"  j/k     move through posts",
		"  enter   open a thread",
		"  c       compose a post",
		"  R       reply · L like · p profile",
		"  a       login (browser cookies)",
		"  t       pick a color theme",
		"  ?       full help · q quit",
		"",
		s.accent.Render("Get started"),
		"  1  enter   browse demo feed",
		"  2  l       login from your browser",
		"  3  t       choose a theme first",
		"",
		s.hint.Render("Or in a terminal:  threadterm login"),
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
			"Same as bird (Twitter CLI): use your browser session.",
			"Be logged into threads.com in Chrome/Firefox first.",
			"",
			s.accent.Render("1 / b") + "  Auto from browser  ← easiest",
			s.accent.Render("2 / w") + "  Username + password",
			s.accent.Render("3 / c") + "  Paste cookies manually",
			s.accent.Render("4 / d") + "  Stay in demo mode",
			"",
			s.muted.Render("Or just run:  threadterm login"),
			s.muted.Render("docs/AUTH.md  ·  esc back"),
		}, "\n")
	case 1:
		body = strings.Join([]string{
			s.section.Render(" COOKIES "),
			"",
			"Optional. Password login is enough for most things.",
			"Paste only if you want better profile search.",
			"",
			m.cookieInput.View(),
			"",
			s.muted.Render("enter save · esc back"),
		}, "\n")
	case 2:
		body = strings.Join([]string{
			s.section.Render(" USERNAME "),
			"",
			"Threads / Instagram username:",
			"",
			m.userInput.View(),
			"",
			s.muted.Render("enter next · esc back"),
		}, "\n")
	case 3:
		body = strings.Join([]string{
			s.section.Render(" PASSWORD "),
			"",
			"Password (hidden):",
			"",
			m.passInput.View(),
			"",
			s.muted.Render("enter login · esc back"),
			s.hint.Render("2FA? use: threadterm login --totp SECRET"),
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

func (m Model) renderSearch() string {
	s := m.styles
	return strings.Join([]string{
		s.section.Render(" SEARCH "),
		"",
		s.muted.Render("Find posts by text query (e.g. lrt kelana jaya)"),
		"",
		m.searchInput.View(),
		"",
		s.hint.Render("enter search · esc cancel"),
		s.muted.Render("CLI: threadterm search malaysia --json"),
	}, "\n")
}
