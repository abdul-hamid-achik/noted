package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"
	"github.com/sahilm/fuzzy"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/muesli/termenv"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/notesync"
	"github.com/abdul-hamid-achik/noted/internal/tui/layout"
	"github.com/abdul-hamid-achik/noted/internal/tui/theme"
)

type editorFocus int

const (
	focusContent editorFocus = iota
	focusTitle
)

type editorMode int

const (
	editorSplit editorMode = iota // content + live preview
	editorEdit                    // content only
)

// noteSavedMsg is emitted after a successful save.
type noteSavedMsg struct{ note db.Note }

type editorView struct {
	title   textinput.Model
	content textarea.Model
	preview viewport.Model

	note     *db.Note // nil when creating
	creating bool
	mode     editorMode
	focus    editorFocus
	dirty    bool
	backArmed bool

	splitOK  bool
	lastArea layout.Rect

	// [[wikilink]] autocomplete state
	acActive  bool
	acOpen    int      // index of the "[[" being completed
	acEnd     int      // index where the query ends (end of value)
	acMatches []string // candidate note titles
	acSel     int
	acTitles  []string // all note titles (loaded lazily per editor session)
	acLoaded  bool
}

func newEditorView() *editorView {
	ti := textinput.New()
	ti.Placeholder = "Note title…"
	ti.Prompt = ""
	theme.TextInput(&ti)

	ta := textarea.New()
	ta.Placeholder = "Start writing… (markdown)"
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	theme.Textarea(&ta)

	vp := viewport.New(viewport.WithWidth(1), viewport.WithHeight(1))
	theme.Viewport(&vp)

	return &editorView{title: ti, content: ta, preview: vp, mode: editorSplit, focus: focusContent}
}

func (v *editorView) id() ViewID         { return ViewEditor }
func (v *editorView) wantsSidebar() bool { return false }
func (v *editorView) capturesText() bool { return true }
func (v *editorView) shortHelp() string {
	return "ctrl+s save · ctrl+l follow [[link]] · ctrl+b backlinks · ctrl+p split/edit · esc back"
}
func (v *editorView) load(a *App) tea.Cmd { return nil }

// set loads a note (or a blank note when creating) into the editor and returns the focus command.
func (v *editorView) set(note db.Note, creating bool) tea.Cmd {
	v.creating = creating
	if creating {
		v.note = nil
	} else {
		n := note
		v.note = &n
	}
	v.title.SetValue(note.Title)
	v.content.SetValue(note.Content)
	v.dirty = false
	v.backArmed = false
	v.mode = editorSplit
	v.acActive = false
	v.acLoaded = false
	v.acTitles = nil
	var cmd tea.Cmd
	if creating {
		cmd = v.setFocus(focusTitle)
	} else {
		cmd = v.setFocus(focusContent)
	}
	v.refreshPreview()
	return cmd
}

func (v *editorView) setFocus(f editorFocus) tea.Cmd {
	v.focus = f
	if f == focusTitle {
		v.content.Blur()
		return v.title.Focus()
	}
	v.title.Blur()
	return v.content.Focus()
}

func (v *editorView) refreshPreview() {
	w := v.preview.Width()
	if w < 10 {
		w = 60
	}
	v.preview.SetContent(renderMarkdown(v.content.Value(), w))
}

func (v *editorView) resize(area layout.Rect) {
	v.lastArea = area
	v.splitOK = area.W >= 72
	taH := max(1, area.H-3)
	if v.mode == editorSplit && v.splitOK {
		cw := (area.W - 3) / 2
		pw := area.W - 3 - cw
		v.content.SetWidth(max(4, cw))
		v.content.SetHeight(taH)
		v.preview.SetWidth(max(4, pw))
		v.preview.SetHeight(taH)
	} else {
		v.content.SetWidth(max(4, area.W))
		v.content.SetHeight(taH)
	}
	v.title.SetWidth(max(8, area.W-8))
	v.refreshPreview()
}

func (v *editorView) save(a *App) tea.Cmd {
	title := strings.TrimSpace(v.title.Value())
	if title == "" {
		title = "Untitled"
	}
	content := v.content.Value()
	creating := v.creating
	var id int64
	var oldTitle, oldContent string
	if v.note != nil {
		id = v.note.ID
		oldTitle, oldContent = v.note.Title, v.note.Content
	}
	ctx, dbq, vlt, w := a.ctx, a.db, a.vlt, a.watcher
	return func() tea.Msg {
		var n db.Note
		var err error
		if creating {
			n, err = dbq.CreateNote(ctx, db.CreateNoteParams{Title: title, Content: content})
		} else {
			// Snapshot the pre-edit state as a version before overwriting it (only when it changed),
			// so TUI edits build the same history as `noted edit` / MCP. The error is intentionally
			// non-fatal here: a versioning hiccup must not block an interactive save (unlike the CLI/
			// MCP paths, which abort) — the edit itself still proceeds below.
			if title != oldTitle || content != oldContent {
				_ = notesync.SnapshotVersion(ctx, dbq, vlt, id, oldTitle, oldContent)
			}
			n, err = dbq.UpdateNote(ctx, db.UpdateNoteParams{ID: id, Title: title, Content: content})
		}
		if err != nil {
			return errMsg{err}
		}
		syncNoteLinks(ctx, dbq, n.ID, content)  // keep [[wikilinks]] / backlinks in sync
		w.PauseSelfWrite()                      // this write is ours — don't let the watcher rebuild on it
		notesync.WriteThrough(ctx, dbq, vlt, n) // mirror the note to the markdown vault
		return noteSavedMsg{note: n}
	}
}

const acMaxMatches = 6

// detectWikilinkQuery finds an in-progress [[query at the end of value: the last "[[" that has no
// closing "]]" and no "]" or newline after it (i.e. the user is actively typing a link).
func detectWikilinkQuery(value string) (open int, end int, query string, ok bool) {
	open = strings.LastIndex(value, "[[")
	if open < 0 {
		return 0, 0, "", false
	}
	after := value[open+2:]
	if strings.ContainsAny(after, "]\n") { // closed, or token ended — not an active typing context
		return 0, 0, "", false
	}
	return open, len(value), after, true
}

func (v *editorView) loadAcTitles(a *App) {
	v.acLoaded = true
	v.acTitles = nil
	if a.db == nil {
		return
	}
	notes, err := a.db.ListNotes(a.ctx, db.ListNotesParams{Limit: 1000, Offset: 0})
	if err != nil {
		return
	}
	for _, n := range notes {
		if t := strings.TrimSpace(n.Title); t != "" {
			v.acTitles = append(v.acTitles, t)
		}
	}
}

func matchTitles(titles []string, query string) []string {
	if strings.TrimSpace(query) == "" {
		if len(titles) > acMaxMatches {
			return titles[:acMaxMatches]
		}
		return titles
	}
	lower := make([]string, len(titles))
	for i, t := range titles {
		lower[i] = strings.ToLower(t)
	}
	var out []string
	for _, m := range fuzzy.Find(strings.ToLower(query), lower) {
		out = append(out, titles[m.Index])
		if len(out) >= acMaxMatches {
			break
		}
	}
	return out
}

// refreshAutocomplete recomputes the [[ autocomplete state from the current content.
func (v *editorView) refreshAutocomplete(a *App) {
	open, end, query, ok := detectWikilinkQuery(v.content.Value())
	if !ok {
		v.acActive = false
		return
	}
	if !v.acLoaded {
		v.loadAcTitles(a)
	}
	wasActive := v.acActive
	v.acActive = true
	v.acOpen, v.acEnd = open, end
	v.acMatches = matchTitles(v.acTitles, query)
	if !wasActive {
		v.acSel = 0
	}
	if v.acSel >= len(v.acMatches) {
		v.acSel = max(0, len(v.acMatches)-1)
	}
}

// acceptAutocomplete replaces the in-progress [[query with [[Title]].
func (v *editorView) acceptAutocomplete() {
	defer func() { v.acActive = false }()
	if len(v.acMatches) == 0 {
		return
	}
	val := v.content.Value()
	if v.acOpen < 0 || v.acEnd > len(val) {
		return
	}
	v.content.SetValue(val[:v.acOpen] + "[[" + v.acMatches[v.acSel] + "]]" + val[v.acEnd:])
	v.dirty = true
	v.backArmed = false
	v.refreshPreview()
}

func (v *editorView) acStrip() string {
	if len(v.acMatches) == 0 {
		return theme.MutedStyle.Render("[[ no matching notes — Esc to dismiss")
	}
	parts := make([]string, len(v.acMatches))
	for i, m := range v.acMatches {
		if i == v.acSel {
			parts[i] = theme.Selected.Render(" " + m + " ")
		} else {
			parts[i] = lipgloss.NewStyle().Foreground(theme.Muted).Render(" " + m + " ")
		}
	}
	return theme.HelpKey.Render("[[ ") + strings.Join(parts, theme.MutedStyle.Render("·")) +
		theme.MutedStyle.Render("  ↹/enter insert · esc dismiss")
}

func (v *editorView) update(a *App, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case noteSavedMsg:
		n := msg.note
		v.note = &n
		v.creating = false
		v.dirty = false
		v.backArmed = false
		a.status = "saved: " + n.Title
		return nil

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			if zone.Get("editor:title").InBounds(msg) {
				return v.setFocus(focusTitle)
			}
			if zone.Get("editor:content").InBounds(msg) {
				return v.setFocus(focusContent)
			}
		}
		return nil

	case tea.MouseWheelMsg:
		if zone.Get("editor:preview").InBounds(msg) {
			switch msg.Button {
			case tea.MouseWheelUp:
				v.preview.ScrollUp(2)
			case tea.MouseWheelDown:
				v.preview.ScrollDown(2)
			}
			return nil
		}
		var cmd tea.Cmd
		v.content, cmd = v.content.Update(msg)
		return cmd

	case tea.KeyMsg:
		// While [[ autocomplete is open it owns navigation / accept / dismiss keys.
		if v.acActive {
			switch msg.String() {
			case "esc":
				v.acActive = false
				return nil
			case "enter", "tab":
				v.acceptAutocomplete()
				return nil
			case "up", "ctrl+p":
				if v.acSel > 0 {
					v.acSel--
				}
				return nil
			case "down", "ctrl+n":
				if v.acSel < len(v.acMatches)-1 {
					v.acSel++
				}
				return nil
			}
		}
		switch msg.String() {
		case "ctrl+s":
			v.backArmed = false
			return v.save(a)
		case "esc":
			if v.dirty && !v.backArmed {
				v.backArmed = true
				a.status = "Unsaved changes — press Esc again to discard"
				return nil
			}
			return a.backToNotes()
		case "ctrl+p":
			if v.mode == editorSplit {
				v.mode = editorEdit
			} else {
				v.mode = editorSplit
			}
			v.resize(v.lastArea)
			return nil
		case "tab":
			if v.focus == focusContent {
				return v.setFocus(focusTitle)
			}
			return v.setFocus(focusContent)
		case "ctrl+l":
			links := parseWikilinks(v.content.Value())
			if len(links) == 0 {
				a.status = "no [[links]] in this note"
				return nil
			}
			ov, cmd := newLinksOverlay(links)
			a.overlay = ov
			return cmd
		case "ctrl+b":
			if v.note == nil || a.db == nil {
				a.status = "save the note first to see backlinks"
				return nil
			}
			back, err := a.db.GetBacklinks(a.ctx, v.note.ID)
			if err != nil || len(back) == 0 {
				a.status = "no backlinks"
				return nil
			}
			ov, cmd := newNoteListOverlay("Backlinks", "Filter backlinks…", back)
			a.overlay = ov
			return cmd
		}

		var cmd tea.Cmd
		if v.focus == focusTitle {
			before := v.title.Value()
			v.title, cmd = v.title.Update(msg)
			if v.title.Value() != before {
				v.dirty = true
				v.backArmed = false
			}
		} else {
			before := v.content.Value()
			v.content, cmd = v.content.Update(msg)
			if v.content.Value() != before {
				v.dirty = true
				v.backArmed = false
				v.refreshPreview()
				v.refreshAutocomplete(a)
			}
		}
		return cmd
	}
	return nil
}

func (v *editorView) render(a *App, area layout.Rect) string {
	titleLabel := theme.MutedStyle.Render("Title ")
	meta := "new"
	if !v.creating && v.note != nil {
		meta = fmt.Sprintf("id:%d", v.note.ID)
	}
	if v.dirty {
		meta += " ●"
	}
	headerInner := lipgloss.JoinHorizontal(lipgloss.Left, titleLabel, v.title.View())
	metaW := lipgloss.Width(meta)
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Width(max(0, area.W-metaW-1)).Render(headerInner),
		theme.MutedStyle.Render(meta),
	)

	var body string
	if v.mode == editorSplit && v.splitOK {
		cw := (area.W - 3) / 2
		pw := area.W - 3 - cw
		cBlock := lipgloss.JoinVertical(lipgloss.Left, paneHeader("EDIT", v.focus == focusContent), v.content.View())
		pBlock := lipgloss.JoinVertical(lipgloss.Left, paneHeader("PREVIEW", false), v.preview.View())
		sep := lipgloss.NewStyle().Foreground(theme.Border).Render(vbar(max(1, area.H-2)))
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			zone.Mark("editor:content", lipgloss.NewStyle().Width(cw).Render(cBlock)),
			" ", sep, " ",
			zone.Mark("editor:preview", lipgloss.NewStyle().Width(pw).Render(pBlock)),
		)
	} else {
		cBlock := lipgloss.JoinVertical(lipgloss.Left, paneHeader("EDIT", true), v.content.View())
		body = zone.Mark("editor:content", lipgloss.NewStyle().Width(area.W).Render(cBlock))
	}

	rows := []string{zone.Mark("editor:title", header)}
	if v.acActive {
		rows = append(rows, v.acStrip())
	}
	rows = append(rows, "", body)
	content := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return lipgloss.NewStyle().Width(area.W).Height(area.H).Render(content)
}

func paneHeader(label string, focused bool) string {
	if focused {
		return theme.HelpKey.Bold(true).Render(label)
	}
	return theme.MutedStyle.Render(label)
}

func vbar(n int) string {
	if n < 1 {
		n = 1
	}
	lines := make([]string, n)
	for i := range lines {
		lines[i] = "│"
	}
	return strings.Join(lines, "\n")
}

func renderMarkdown(content string, width int) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(theme.GlamourJSON),
		glamour.WithColorProfile(termenv.TrueColor),
		glamour.WithEmoji(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return content
	}
	out, err := r.Render(content)
	if err != nil || strings.TrimSpace(out) == "" {
		return content
	}
	return out
}
