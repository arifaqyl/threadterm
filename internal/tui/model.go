package tui

import (
	"github.com/arifaqyl/threadterm/internal/api"
	"github.com/arifaqyl/threadterm/internal/config"
	"github.com/arifaqyl/threadterm/internal/models"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	viewSearch
)

type feedLoadedMsg struct {
	page  models.FeedPage
	err   error
	query string
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

	posts  []models.Post
	cursor int
	// feedOffsets[i] = starting line of post i inside the viewport content.
	feedOffsets []int
	feedSource  string
	feedHint    string
	thread      *models.Thread
	profile     *models.User
	pPosts      []models.Post
	loading     bool

	compose textarea.Model
	replyTo string

	searchInput textinput.Model
	lastQuery   string

	// login form
	// 0=menu 1=cookies 2=user 3=pass 4=official-token 5=official-uid 6=oauth
	loginStep   int
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

	si := textinput.New()
	si.Placeholder = "search users / topics…"
	si.CharLimit = 80
	si.Width = 48

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
		searchInput: si,
		status:      "ready",
		loading:     start == viewFeed,
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

// Run starts the Bubble Tea program.
func Run(cfg *config.Config, client api.Client) error {
	// No mouse capture — so you can select/copy text in the terminal normally.
	p := tea.NewProgram(
		New(cfg, client),
		tea.WithAltScreen(),
	)
	_, err := p.Run()
	return err
}
