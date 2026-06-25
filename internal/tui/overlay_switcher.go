package tui

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/tui/theme"
)

const (
	switcherZonePrefix = "sw:"
	switcherMaxRows    = 12
)

// switcherOverlay is the Ctrl+O quick switcher: fuzzy jump-to-note. Mirrors the palette overlay
// pattern but lists notes from the db and opens the chosen one in the editor.
type switcherOverlay struct {
	title    string
	input    textinput.Model
	notes    []db.Note
	filtered []int
	cursor   int
}

func newSwitcher(notes []db.Note) (*switcherOverlay, tea.Cmd) {
	return newNoteListOverlay("Jump to Note", "Jump to note…", notes)
}

// newNoteListOverlay is a fuzzy note picker reused by the quick switcher and the backlinks panel.
func newNoteListOverlay(title, placeholder string, notes []db.Note) (*switcherOverlay, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = "❯ "
	theme.TextInput(&ti)
	cmd := ti.Focus()

	s := &switcherOverlay{title: title, input: ti, notes: notes}
	s.refilter()
	return s, cmd
}

func (s *switcherOverlay) refilter() {
	q := strings.TrimSpace(s.input.Value())
	s.filtered = s.filtered[:0]
	if q == "" {
		for i := range s.notes {
			s.filtered = append(s.filtered, i)
		}
	} else {
		titles := make([]string, len(s.notes))
		for i, n := range s.notes {
			titles[i] = strings.ToLower(noteItem{note: n}.Title())
		}
		for _, m := range fuzzy.Find(strings.ToLower(q), titles) {
			s.filtered = append(s.filtered, m.Index)
		}
	}
	if s.cursor >= len(s.filtered) {
		s.cursor = max(0, len(s.filtered)-1)
	}
}

func (s *switcherOverlay) update(a *App, msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+o":
			return true, nil
		case "enter":
			if s.cursor >= 0 && s.cursor < len(s.filtered) {
				return true, a.openEditor(s.notes[s.filtered[s.cursor]], false)
			}
			return true, nil
		case "up", "ctrl+p":
			if s.cursor > 0 {
				s.cursor--
			}
			return false, nil
		case "down", "ctrl+n":
			if s.cursor < len(s.filtered)-1 {
				s.cursor++
			}
			return false, nil
		}
		before := s.input.Value()
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		if s.input.Value() != before {
			s.refilter()
		}
		return false, cmd

	case tea.MouseClickMsg:
		if vi := hitListZone(switcherZonePrefix, len(s.filtered), msg); vi >= 0 {
			return true, a.openEditor(s.notes[s.filtered[vi]], false)
		}
		return false, nil
	}
	// Forward anything else (e.g. cursor-blink ticks) to the input so it keeps blinking.
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return false, cmd
}

func (s *switcherOverlay) render(width, height int) string {
	w := min(70, width-4)
	if w < 20 {
		w = max(10, width-2)
	}

	// Window the visible rows around the cursor.
	start := 0
	if s.cursor >= switcherMaxRows {
		start = s.cursor - switcherMaxRows + 1
	}
	end := min(len(s.filtered), start+switcherMaxRows)

	var b strings.Builder
	b.WriteString(s.input.View())
	b.WriteString("\n\n")
	if len(s.filtered) == 0 {
		b.WriteString(theme.MutedStyle.Render("no matching notes"))
	} else {
		for vi := start; vi < end; vi++ {
			title := noteItem{note: s.notes[s.filtered[vi]]}.Title()
			var line string
			if vi == s.cursor {
				line = theme.Selected.Render("❯ " + title)
			} else {
				line = lipgloss.NewStyle().Foreground(theme.Text).Render("  " + title)
			}
			b.WriteString(zone.Mark(switcherZonePrefix+strconv.Itoa(vi), line))
			b.WriteString("\n")
		}
	}

	modal := theme.Modal.Width(w).Render(
		lipgloss.JoinVertical(lipgloss.Left, theme.Title.Render(s.title), "", b.String()),
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal)
}
