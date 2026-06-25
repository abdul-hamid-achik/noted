package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func TestHitListZone(t *testing.T) {
	zone.NewGlobal() // so zone.Get has a manager; with no Scan, every zone is out of bounds

	// A non-left button never registers a hit (and short-circuits before any zone lookup).
	if got := hitListZone("pal:", 5, tea.MouseClickMsg{Button: tea.MouseRight}); got != -1 {
		t.Errorf("right-click = %d, want -1", got)
	}
	// A left click with no marked/scanned rows misses.
	if got := hitListZone("pal:", 5, tea.MouseClickMsg{Button: tea.MouseLeft}); got != -1 {
		t.Errorf("left-click with no zones = %d, want -1", got)
	}
	// Zero rows: nothing to hit.
	if got := hitListZone("pal:", 0, tea.MouseClickMsg{Button: tea.MouseLeft}); got != -1 {
		t.Errorf("zero rows = %d, want -1", got)
	}
}
