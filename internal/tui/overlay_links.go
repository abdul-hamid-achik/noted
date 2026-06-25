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

const linksZonePrefix = "lnk:"

// linksOverlay lets you follow a [[wikilink]] from the current note: pick a link, and it opens the
// matching note (creating it if the title doesn't exist yet — Obsidian behavior).
type linksOverlay struct {
	input    textinput.Model
	links    []string
	filtered []int
	cursor   int
}

func newLinksOverlay(links []string) (*linksOverlay, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "Filter links…"
	ti.Prompt = "❯ "
	theme.TextInput(&ti)
	cmd := ti.Focus()

	o := &linksOverlay{input: ti, links: links}
	o.refilter()
	return o, cmd
}

func (o *linksOverlay) refilter() {
	q := strings.TrimSpace(o.input.Value())
	o.filtered = o.filtered[:0]
	if q == "" {
		for i := range o.links {
			o.filtered = append(o.filtered, i)
		}
	} else {
		lower := make([]string, len(o.links))
		for i, l := range o.links {
			lower[i] = strings.ToLower(l)
		}
		for _, m := range fuzzy.Find(strings.ToLower(q), lower) {
			o.filtered = append(o.filtered, m.Index)
		}
	}
	if o.cursor >= len(o.filtered) {
		o.cursor = max(0, len(o.filtered)-1)
	}
}

func (o *linksOverlay) follow(a *App, title string) tea.Cmd {
	if a.db != nil {
		if note, err := a.db.GetNoteByTitle(a.ctx, title); err == nil {
			return a.openEditor(note, false)
		}
	}
	// Unresolved link → create the note (title prefilled).
	return a.openEditor(db.Note{Title: title}, true)
}

func (o *linksOverlay) update(a *App, msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return true, nil
		case "enter":
			if o.cursor >= 0 && o.cursor < len(o.filtered) {
				return true, o.follow(a, o.links[o.filtered[o.cursor]])
			}
			return true, nil
		case "up", "ctrl+p":
			if o.cursor > 0 {
				o.cursor--
			}
			return false, nil
		case "down", "ctrl+n":
			if o.cursor < len(o.filtered)-1 {
				o.cursor++
			}
			return false, nil
		}
		before := o.input.Value()
		var cmd tea.Cmd
		o.input, cmd = o.input.Update(msg)
		if o.input.Value() != before {
			o.refilter()
		}
		return false, cmd

	case tea.MouseClickMsg:
		if vi := hitListZone(linksZonePrefix, len(o.filtered), msg); vi >= 0 {
			return true, o.follow(a, o.links[o.filtered[vi]])
		}
		return false, nil
	}
	// Forward anything else (e.g. cursor-blink ticks) to the input so it keeps blinking.
	var cmd tea.Cmd
	o.input, cmd = o.input.Update(msg)
	return false, cmd
}

func (o *linksOverlay) render(width, height int) string {
	w := min(60, width-4)
	if w < 20 {
		w = max(10, width-2)
	}

	var b strings.Builder
	b.WriteString(o.input.View())
	b.WriteString("\n\n")
	if len(o.filtered) == 0 {
		b.WriteString(theme.MutedStyle.Render("no links"))
	} else {
		for vi, li := range o.filtered {
			label := "[[" + o.links[li] + "]]"
			var line string
			if vi == o.cursor {
				line = theme.Selected.Render("❯ " + label)
			} else {
				line = lipgloss.NewStyle().Foreground(theme.Link).Render("  " + label)
			}
			b.WriteString(zone.Mark(linksZonePrefix+strconv.Itoa(vi), line))
			b.WriteString("\n")
		}
	}

	modal := theme.Modal.Width(w).Render(
		lipgloss.JoinVertical(lipgloss.Left, theme.Title.Render("Follow Link"), "", b.String()),
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal)
}
