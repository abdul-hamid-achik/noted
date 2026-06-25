package tui

import (
	"strconv"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

// hitListZone returns the index of the list row whose bubblezone a left-click landed in, or -1 for a
// non-left click or a miss. Overlays mark their rows as "<prefix><index>"; this is the shared inverse
// used by the palette, switcher, and links overlays so their mouse hit-testing stays identical (and
// coordinate-free — see CLAUDE.md "Don't hardcode coordinates for mouse").
func hitListZone(prefix string, count int, msg tea.MouseClickMsg) int {
	if msg.Button != tea.MouseLeft {
		return -1
	}
	for i := 0; i < count; i++ {
		if zone.Get(prefix + strconv.Itoa(i)).InBounds(msg) {
			return i
		}
	}
	return -1
}
