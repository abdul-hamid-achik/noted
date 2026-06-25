package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/abdul-hamid-achik/noted/internal/notesync"
)

// vaultSyncedMsg is sent after the vault changed on disk and the index was rebuilt, so the active
// view can refresh from the new data.
type vaultSyncedMsg struct{}

// startVaultWatcher begins watching the vault directory for external .md changes. When files change
// (an agent writing a note, $EDITOR, Obsidian, git pull), it rebuilds the index from the vault and
// tells the program to refresh via vaultSyncedMsg. Best-effort: if the watcher can't start (no vault,
// no db handle, or an OS error), live sync is simply disabled and the TUI works as before.
func (a *App) startVaultWatcher(p *tea.Program) {
	if a.vlt == nil || a.conn == nil {
		return
	}
	w, err := notesync.NewWatcher(a.vlt.Dir(), 400*time.Millisecond, func() {
		if _, err := notesync.Rebuild(a.ctx, a.conn, a.vlt); err != nil {
			return
		}
		p.Send(vaultSyncedMsg{})
	})
	if err != nil {
		return
	}
	a.watcher = w
}

// closeWatcher stops the vault watcher (idempotent).
func (a *App) closeWatcher() {
	if a.watcher != nil {
		_ = a.watcher.Close()
		a.watcher = nil
	}
}
