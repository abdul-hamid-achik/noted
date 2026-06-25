package tui

import (
	"context"
	"testing"
)

func TestDetectWikilinkQuery(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantOK    bool
		wantQuery string
		wantOpen  int
	}{
		{"no brackets", "just some text", false, "", 0},
		{"actively typing", "see [[Pro", true, "Pro", 4},
		{"empty query right after open", "intro [[", true, "", 6},
		{"closed link is not active", "see [[Done]]", false, "", 0},
		{"single closing bracket cancels", "see [[Done]", false, "", 0},
		{"newline cancels", "see [[Multi\nline", false, "", 0},
		{"uses the last open link", "[[A]] and now [[Be", true, "Be", 14},
		{"query keeps spaces", "[[Meeting no", true, "Meeting no", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			open, end, query, ok := detectWikilinkQuery(tt.value)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if query != tt.wantQuery {
				t.Errorf("query = %q, want %q", query, tt.wantQuery)
			}
			if open != tt.wantOpen {
				t.Errorf("open = %d, want %d", open, tt.wantOpen)
			}
			if end != len(tt.value) {
				t.Errorf("end = %d, want %d (len)", end, len(tt.value))
			}
		})
	}
}

func TestMatchTitles(t *testing.T) {
	titles := []string{"Project ideas", "Meeting notes", "Reading list", "Roadmap"}

	// Empty query returns everything (when under the cap).
	if got := matchTitles(titles, ""); len(got) != len(titles) {
		t.Errorf("empty query = %d matches, want %d", len(got), len(titles))
	}

	// Empty query is capped at acMaxMatches.
	many := make([]string, acMaxMatches+5)
	for i := range many {
		many[i] = string(rune('A' + i))
	}
	if got := matchTitles(many, ""); len(got) != acMaxMatches {
		t.Errorf("empty query over cap = %d, want %d", len(got), acMaxMatches)
	}

	// Fuzzy match is case-insensitive.
	got := matchTitles(titles, "proj")
	if len(got) == 0 || got[0] != "Project ideas" {
		t.Errorf("query %q = %v, want first match \"Project ideas\"", "proj", got)
	}

	// No match returns empty.
	if got := matchTitles(titles, "zzzzz"); len(got) != 0 {
		t.Errorf("no-match query = %v, want empty", got)
	}
}

// TestEveryNavItemResolvesToView locks the invariant that the placeholder fallback guarantees:
// every sidebar nav entry maps to a non-nil View (so navigating can never hit a nil model).
func TestEveryNavItemResolvesToView(t *testing.T) {
	a := newApp(context.Background(), nil, nil, nil)
	for _, n := range navItems {
		if v := a.views[n.id]; v == nil {
			t.Errorf("nav item %q (id %v) has no registered view", n.label, n.id)
		}
	}
	if a.activeView() == nil {
		t.Fatal("active view is nil on a fresh App")
	}
}
