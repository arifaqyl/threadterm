package tui

import "github.com/charmbracelet/lipgloss"

// Theme is a named color palette for the TUI.
type Theme struct {
	Name       string
	BrandFg    string
	BrandBg    string
	Accent     string
	Muted      string
	Error      string
	Tag        string
	Border     string
	SelectBg   string
	SelectBord string
	PanelBg    string
	Text       string
}

var themes = []Theme{
	{
		Name: "jade", BrandFg: "#0A0A0A", BrandBg: "#E8E6E1", Accent: "#3D9B7A",
		Muted: "#7A7A72", Error: "#C45C4A", Tag: "#C4A35A", Border: "#2A2A28",
		SelectBg: "#141816", SelectBord: "#3D9B7A", PanelBg: "#101210", Text: "#E8E6E1",
	},
	{
		Name: "ocean", BrandFg: "#F5F8FA", BrandBg: "#0B3D5C", Accent: "#3BA3D9",
		Muted: "#7A93A3", Error: "#E06B6B", Tag: "#5EC4B6", Border: "#1A3A4A",
		SelectBg: "#0E2433", SelectBord: "#3BA3D9", PanelBg: "#0A1A24", Text: "#E8F1F6",
	},
	{
		Name: "ember", BrandFg: "#1A1008", BrandBg: "#F0E0C8", Accent: "#D9773A",
		Muted: "#8A7A68", Error: "#C04040", Tag: "#E0A040", Border: "#3A2A1A",
		SelectBg: "#1C140E", SelectBord: "#D9773A", PanelBg: "#14100C", Text: "#F0E6D8",
	},
	{
		Name: "mono", BrandFg: "#111111", BrandBg: "#EEEEEE", Accent: "#FFFFFF",
		Muted: "#888888", Error: "#FF6666", Tag: "#CCCCCC", Border: "#444444",
		SelectBg: "#1A1A1A", SelectBord: "#FFFFFF", PanelBg: "#0D0D0D", Text: "#EEEEEE",
	},
	{
		Name: "orchid", BrandFg: "#1A0A18", BrandBg: "#F0E0EC", Accent: "#C45A9A",
		Muted: "#8A7080", Error: "#D05050", Tag: "#A070C0", Border: "#3A2030",
		SelectBg: "#180E16", SelectBord: "#C45A9A", PanelBg: "#120A10", Text: "#F0E6EC",
	},
}

func themeByName(name string) Theme {
	for _, t := range themes {
		if t.Name == name {
			return t
		}
	}
	return themes[0]
}

func nextTheme(name string) Theme {
	for i, t := range themes {
		if t.Name == name {
			return themes[(i+1)%len(themes)]
		}
	}
	return themes[0]
}

func themeNames() []string {
	out := make([]string, len(themes))
	for i, t := range themes {
		out[i] = t.Name
	}
	return out
}

type styles struct {
	brand    lipgloss.Style
	mode     lipgloss.Style
	accent   lipgloss.Style
	muted    lipgloss.Style
	err      lipgloss.Style
	tag      lipgloss.Style
	section  lipgloss.Style
	footer   lipgloss.Style
	card     lipgloss.Style
	selected lipgloss.Style
	compose  lipgloss.Style
	sidebar  lipgloss.Style
	panel    lipgloss.Style
	hint     lipgloss.Style
	title    lipgloss.Style
}

func makeStyles(th Theme) styles {
	return styles{
		brand: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.BrandFg)).Background(lipgloss.Color(th.BrandBg)),
		mode:  lipgloss.NewStyle().Foreground(lipgloss.Color(th.BrandBg)).Background(lipgloss.Color(th.Accent)),
		accent: lipgloss.NewStyle().Foreground(lipgloss.Color(th.Accent)).Bold(true),
		muted:  lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted)),
		err:    lipgloss.NewStyle().Foreground(lipgloss.Color(th.Error)),
		tag:    lipgloss.NewStyle().Foreground(lipgloss.Color(th.Tag)),
		section: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.BrandFg)).Background(lipgloss.Color(th.Accent)),
		footer:  lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted)),
		card: lipgloss.NewStyle().Padding(0, 1).MarginLeft(0).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color(th.Border)),
		selected: lipgloss.NewStyle().Padding(0, 1).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color(th.SelectBord)).
			Background(lipgloss.Color(th.SelectBg)),
		compose: lipgloss.NewStyle().Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(th.Accent)),
		sidebar: lipgloss.NewStyle().Padding(0, 1).
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color(th.Border)),
		panel: lipgloss.NewStyle().Padding(1, 2),
		hint:  lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted)).Italic(true),
		title: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.Text)),
	}
}
