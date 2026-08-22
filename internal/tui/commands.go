package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/arifaqyl/threadterm/internal/api"
	"github.com/arifaqyl/threadterm/internal/auth"
	"github.com/arifaqyl/threadterm/internal/config"
	"github.com/arifaqyl/threadterm/internal/models"
)

func loadFeed(c api.Client) tea.Cmd {
	return func() tea.Msg {
		page, err := c.Feed("", 40)
		return feedLoadedMsg{page: page, err: err}
	}
}

func loadDiscover(c api.Client) tea.Cmd {
	return func() tea.Msg {
		page, err := c.Discover(40)
		return feedLoadedMsg{page: page, err: err}
	}
}

func loadSearch(c api.Client, q string) tea.Cmd {
	return func() tea.Msg {
		page, err := c.Search(q, 40)
		return feedLoadedMsg{page: page, err: err, query: q}
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
