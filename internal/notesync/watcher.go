package notesync

import (
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches a vault directory for markdown changes and invokes a callback (debounced) so the
// index can be rebuilt when files change outside the app — an agent writing a .md, an external
// $EDITOR, Obsidian, or `git pull`. Best-effort: a failure to start is reported to the caller, and
// runtime errors are swallowed rather than crashing the UI.
type Watcher struct {
	fw       *fsnotify.Watcher
	debounce time.Duration
	onChange func()
	done     chan struct{}

	mu         sync.Mutex
	mutedUntil time.Time // ignore events until this time (set around the app's own writes)
}

// NewWatcher starts watching dir for .md create/write/remove/rename events. onChange fires once per
// quiet window (debounce) after activity settles — editors emit several events per save, so a single
// debounced callback avoids redundant rebuilds. Call Close to stop. A zero/negative debounce defaults
// to 300ms.
func NewWatcher(dir string, debounce time.Duration, onChange func()) (*Watcher, error) {
	if debounce <= 0 {
		debounce = 300 * time.Millisecond
	}
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := fw.Add(dir); err != nil {
		_ = fw.Close()
		return nil, err
	}
	w := &Watcher{fw: fw, debounce: debounce, onChange: onChange, done: make(chan struct{})}
	go w.loop()
	return w, nil
}

// relevant reports whether an event touches a markdown file we care about.
func relevantEvent(ev fsnotify.Event) bool {
	if !strings.HasSuffix(strings.ToLower(ev.Name), ".md") {
		return false
	}
	// Create/Write/Remove/Rename all change the vault's contents; Chmod alone does not.
	return ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) != 0
}

func (w *Watcher) loop() {
	var timer *time.Timer
	var fire <-chan time.Time
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		select {
		case <-w.done:
			return
		case ev, ok := <-w.fw.Events:
			if !ok {
				return
			}
			if !relevantEvent(ev) {
				continue
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(w.debounce)
			fire = timer.C
		case <-fire:
			fire = nil
			if w.muted() { // the change was the app's own write — DB already matches, skip the rebuild
				continue
			}
			if w.onChange != nil {
				w.onChange()
			}
		case _, ok := <-w.fw.Errors:
			if !ok {
				return
			}
		}
	}
}

// PauseSelfWrite mutes the watcher briefly so the app's own vault writes (write-through, delete)
// don't trigger a redundant index rebuild — the DB already matches those changes. The window covers
// the debounce plus margin, so the self-write's event is dropped while genuine external edits after it
// still fire. Safe to call from any goroutine; nil-safe.
func (w *Watcher) PauseSelfWrite() {
	if w == nil {
		return
	}
	until := time.Now().Add(2*w.debounce + 400*time.Millisecond)
	w.mu.Lock()
	if until.After(w.mutedUntil) {
		w.mutedUntil = until
	}
	w.mu.Unlock()
}

func (w *Watcher) muted() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return time.Now().Before(w.mutedUntil)
}

// Close stops the watcher and releases its OS resources. Safe to call once.
func (w *Watcher) Close() error {
	if w == nil {
		return nil
	}
	close(w.done)
	return w.fw.Close()
}
