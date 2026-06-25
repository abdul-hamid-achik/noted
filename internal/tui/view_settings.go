package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/abdul-hamid-achik/noted/internal/config"
	"github.com/abdul-hamid-achik/noted/internal/tui/layout"
	"github.com/abdul-hamid-achik/noted/internal/tui/theme"
)

type settingsInfo struct {
	dbPath      string
	vaultPath   string
	veclitePath string
}

type settingsLoadedMsg struct{ info settingsInfo }

type settingsView struct {
	info   settingsInfo
	loaded bool
}

func newSettingsView() *settingsView { return &settingsView{} }

func (v *settingsView) id() ViewID            { return ViewSettings }
func (v *settingsView) wantsSidebar() bool    { return true }
func (v *settingsView) capturesText() bool    { return false }
func (v *settingsView) shortHelp() string     { return "esc back" }
func (v *settingsView) resize(area layout.Rect) {}

func (v *settingsView) load(a *App) tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return errMsg{err}
		}
		return settingsLoadedMsg{info: settingsInfo{
			dbPath:      cfg.DBPath,
			vaultPath:   cfg.VaultPath,
			veclitePath: cfg.VeclitePath,
		}}
	}
}

func (v *settingsView) update(a *App, msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(settingsLoadedMsg); ok {
		v.info = msg.info
		v.loaded = true
		return nil
	}
	if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "esc" {
		return a.backToNotes()
	}
	return nil
}

func (v *settingsView) render(a *App, area layout.Rect) string {
	row := func(k, val string) string {
		return lipgloss.JoinHorizontal(lipgloss.Left,
			theme.MutedStyle.Width(14).Render(k),
			lipgloss.NewStyle().Foreground(theme.Text).Render(val),
		)
	}
	semantic := "disabled"
	if v.info.veclitePath != "" {
		semantic = "enabled"
	}
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		theme.Heading.Render("Settings"),
		"",
		row("Theme", "Nord (dark)"),
		row("Terminal", "optimized for Ghostty (truecolor)"),
		row("Database", v.info.dbPath),
		row("Vault", v.info.vaultPath+"  (markdown vault — in progress)"),
		row("Semantic", semantic),
		"",
		theme.Subheading.Render("Keys"),
		theme.MutedStyle.Render("1-9 switch view · Ctrl+K palette · Ctrl+O switcher · Tab sidebar · ? help · q quit"),
		theme.MutedStyle.Render("notes: n new · / filter · enter open    editor: Ctrl+S save · Ctrl+L follow [[link]]"),
	)
	return lipgloss.NewStyle().Width(area.W).Height(area.H).Padding(1, 2).Render(body)
}
