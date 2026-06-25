// Package tui is noted's terminal UI. The root App model owns the window size, the active view,
// global keybindings and the command/overlay layer; each view (see view.go) owns its own state.
// Layout regions come from internal/tui/layout; colors/components from internal/tui/theme; mouse
// hit-testing from bubblezone. See docs/dev/charm-v2-reference.md.
package tui

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/notesync"
	"github.com/abdul-hamid-achik/noted/internal/tui/layout"
	"github.com/abdul-hamid-achik/noted/internal/tui/theme"
	"github.com/abdul-hamid-achik/noted/internal/vault"
)

// errMsg carries an async error to the root, shown in the status bar.
type errMsg struct{ err error }

// overlay is a modal layer (command palette, quick switcher) rendered over the active view. While an
// overlay is open it receives all keys and clicks; update returns closed=true to dismiss it.
type overlay interface {
	update(a *App, msg tea.Msg) (closed bool, cmd tea.Cmd)
	render(width, height int) string
}

// App is the root bubbletea model.
type App struct {
	ctx  context.Context
	db   *db.Queries
	conn *sql.DB      // raw handle for vault re-index (nil = no live sync)
	vlt  *vault.Vault // nil = vault write-through disabled

	width  int
	height int
	ready  bool

	active        ViewID
	views         map[ViewID]View
	order         []ViewID
	sidebarForced bool

	overlay overlay          // non-nil when a modal overlay (palette / switcher) is open
	status  string
	watcher *notesync.Watcher // nil = no vault file watching
}

// New builds the bubbletea program for the noted TUI. vlt may be nil to disable vault write-through;
// conn may be nil to disable live re-index on external vault changes.
func New(ctx context.Context, conn *sql.DB, database *db.Queries, vlt *vault.Vault) (*tea.Program, error) {
	zone.NewGlobal()
	a := newApp(ctx, conn, database, vlt)
	p := tea.NewProgram(a)
	a.startVaultWatcher(p)
	return p, nil
}

func newApp(ctx context.Context, conn *sql.DB, database *db.Queries, vlt *vault.Vault) *App {
	a := &App{
		ctx:    ctx,
		db:     database,
		conn:   conn,
		vlt:    vlt,
		width:  80,
		height: 24,
		active: ViewNotes,
		views:  make(map[ViewID]View),
	}
	a.views[ViewNotes] = newNotesView()
	a.views[ViewEditor] = newEditorView()
	a.views[ViewSearch] = newSearchView()
	a.views[ViewTags] = newTagsView()
	a.views[ViewFolders] = newFoldersView()
	a.views[ViewTasks] = newTasksView()
	a.views[ViewDashboard] = newDashboardView()
	a.views[ViewDaily] = newDailyView()
	a.views[ViewTemplates] = newTemplatesView()
	a.views[ViewSettings] = newSettingsView()
	for _, n := range navItems {
		if _, ok := a.views[n.id]; !ok {
			a.views[n.id] = newPlaceholder(n.id, n.label)
		}
		a.order = append(a.order, n.id)
	}
	return a
}

func (a *App) activeView() View { return a.views[a.active] }

func (a *App) regions() layout.Regions {
	return layout.Compute(a.width, a.height, layout.Options{
		ShowSidebar:   a.activeView().wantsSidebar(),
		SidebarForced: a.sidebarForced,
	})
}

func (a *App) editor() *editorView { return a.views[ViewEditor].(*editorView) }

// openEditor switches to the editor for an existing note (creating=false) or a new note.
func (a *App) openEditor(note db.Note, creating bool) tea.Cmd {
	cmd := a.editor().set(note, creating)
	a.active = ViewEditor
	a.status = ""
	a.resizeActive()
	return cmd
}

// backToNotes leaves the current view and reloads the notes list (to reflect saves).
func (a *App) backToNotes() tea.Cmd {
	a.active = ViewNotes
	a.status = ""
	a.resizeActive()
	return a.views[ViewNotes].load(a)
}

// openTag shows the notes list filtered to a single tag.
func (a *App) openTag(name string) tea.Cmd {
	nv := a.views[ViewNotes].(*notesView)
	nv.setFilter(notesFilter{tag: name})
	a.active = ViewNotes
	a.status = "filtering by #" + name + " — esc to clear"
	a.resizeActive()
	return nv.load(a)
}

// openFolder shows the notes list filtered to a single folder.
func (a *App) openFolder(f db.Folder) tea.Cmd {
	nv := a.views[ViewNotes].(*notesView)
	nv.setFilter(notesFilter{folder: &f})
	a.active = ViewNotes
	a.status = "folder: " + f.Name + " — esc to clear"
	a.resizeActive()
	return nv.load(a)
}

func (a *App) resizeActive() {
	r := a.regions()
	if r.Mode == layout.ModeTooSmall {
		return
	}
	a.activeView().resize(r.Main)
}

func (a *App) switchTo(id ViewID) tea.Cmd {
	if a.active == id {
		return nil
	}
	a.active = id
	a.status = ""
	a.resizeActive()
	return a.activeView().load(a)
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	return a.activeView().load(a)
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// The vault changed on disk and was re-indexed: refresh the active view from the new data. Handled
	// before the overlay check so a background sync isn't swallowed while a palette/switcher is open.
	if _, ok := msg.(vaultSyncedMsg); ok {
		a.status = "↻ vault synced"
		return a, a.activeView().load(a)
	}

	// When an overlay (palette / switcher / links) is open it owns input. A window resize still
	// updates the app dimensions; everything else (keys, clicks, cursor-blink ticks) goes to the
	// overlay so its input keeps working and nothing leaks to the background view.
	if a.overlay != nil {
		if ws, ok := msg.(tea.WindowSizeMsg); ok {
			a.width, a.height = ws.Width, ws.Height
			a.ready = true
			a.resizeActive()
			return a, nil
		}
		closed, cmd := a.overlay.update(a, msg)
		if closed {
			a.overlay = nil
		}
		return a, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.ready = true
		a.resizeActive()
		return a, nil

	case errMsg:
		a.status = "error: " + msg.err.Error()
		return a, nil

	case tea.KeyMsg:
		if cmd, handled := a.handleGlobalKey(msg); handled {
			return a, cmd
		}
		return a, a.activeView().update(a, msg)

	case tea.MouseClickMsg:
		return a, a.handleMouseClick(msg)

	case tea.MouseWheelMsg:
		return a, a.activeView().update(a, msg)

	case tea.MouseMotionMsg, tea.MouseReleaseMsg:
		return a, nil

	default:
		return a, a.activeView().update(a, msg)
	}
}

func (a *App) handleGlobalKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	s := msg.String()
	switch s {
	case "ctrl+c":
		a.closeWatcher()
		return tea.Quit, true
	case "q":
		if a.activeView().capturesText() {
			return nil, false
		}
		a.closeWatcher()
		return tea.Quit, true
	case "tab", "\\":
		if a.activeView().capturesText() {
			return nil, false
		}
		a.sidebarForced = !a.sidebarForced
		a.resizeActive()
		return nil, true
	case "ctrl+k":
		// Return the input's focus/blink command so bubbletea emits a follow-up frame (the cursor
		// also blinks); without a command the freshly-opened overlay isn't flushed until the next event.
		ov, cmd := newPalette()
		a.overlay = ov
		return cmd, true
	case "ctrl+o":
		var notes []db.Note
		if a.db != nil {
			notes, _ = a.db.ListNotes(a.ctx, db.ListNotesParams{Limit: 500, Offset: 0})
		}
		ov, cmd := newSwitcher(notes)
		a.overlay = ov
		return cmd, true
	case "?":
		if a.activeView().capturesText() {
			return nil, false
		}
		a.status = a.activeView().shortHelp()
		return nil, true
	}
	// Digits 1-9 jump to sidebar views by index (skipped while a view captures text input).
	if len(s) == 1 && s[0] >= '1' && s[0] <= '9' && !a.activeView().capturesText() {
		if idx := int(s[0] - '1'); idx < len(a.order) {
			return a.switchTo(a.order[idx]), true
		}
	}
	return nil, false
}

func (a *App) handleMouseClick(msg tea.MouseClickMsg) tea.Cmd {
	if msg.Button == tea.MouseLeft {
		for _, id := range a.order {
			if zone.Get(navZoneID(id)).InBounds(msg) {
				return a.switchTo(id)
			}
		}
	}
	return a.activeView().update(a, msg)
}

// View implements tea.Model.
func (a *App) View() tea.View {
	var content string
	switch {
	case !a.ready:
		content = theme.MutedStyle.Render("Loading noted…")
	default:
		r := a.regions()
		switch {
		case r.Mode == layout.ModeTooSmall:
			content = a.renderTooSmall()
		case a.overlay != nil:
			content = a.overlay.render(a.width, a.height)
		default:
			content = a.renderBody(r)
		}
	}

	v := tea.NewView(zone.Scan(content))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.BackgroundColor = theme.Bg
	v.WindowTitle = "noted"
	return v
}

func (a *App) renderBody(r layout.Regions) string {
	main := a.activeView().render(a, r.Main)

	var row string
	if r.SidebarVisible {
		row = lipgloss.JoinHorizontal(lipgloss.Top, a.renderSidebar(r.Sidebar), main)
	} else {
		row = main
	}
	return lipgloss.JoinVertical(lipgloss.Left, row, a.renderFooter(r.Footer))
}

func (a *App) renderSidebar(area layout.Rect) string {
	var b strings.Builder
	b.WriteString(theme.Title.Render("noted"))
	b.WriteString("\n\n")
	for _, id := range a.order {
		var line string
		if id == a.active {
			line = theme.Selected.Render("▸ " + navLabel(id))
		} else {
			line = lipgloss.NewStyle().Foreground(theme.Muted).Render("  " + navLabel(id))
		}
		b.WriteString(zone.Mark(navZoneID(id), line))
		b.WriteString("\n")
	}
	// No top padding so the "noted" brand sits on row 0, aligned with the active view's title.
	return theme.Panel.Width(area.W).Height(area.H).Padding(0, 1).Render(b.String())
}

func (a *App) renderFooter(area layout.Rect) string {
	left := a.status
	if left == "" {
		left = a.activeView().shortHelp()
	}
	max := area.W - 2
	if max < 0 {
		max = 0
	}
	if r := []rune(left); len(r) > max { // rune-safe truncation (status may contain note titles)
		if max > 1 {
			left = string(r[:max-1]) + "…"
		} else if max == 1 {
			left = "…"
		} else {
			left = ""
		}
	}
	return theme.Footer.Width(area.W).Padding(0, 1).Render(left)
}

func (a *App) renderTooSmall() string {
	msg := fmt.Sprintf("Terminal too small\n%d×%d — need at least %d×%d",
		a.width, a.height, layout.MinWidth, layout.MinHeight)
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center,
		theme.Heading.Render(msg))
}
