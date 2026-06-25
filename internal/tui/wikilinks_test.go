package tui

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/abdul-hamid-achik/noted/internal/db"
)

func newTestQueries(t *testing.T) (*db.Queries, context.Context) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return db.New(conn), context.Background()
}

func TestParseWikilinks(t *testing.T) {
	got := parseWikilinks("see [[Alpha]], [[Beta|alias]] and [[Alpha]] again, [[  ]] empty, [[Gamma]]")
	want := []string{"Alpha", "Beta", "Gamma"}
	if len(got) != len(want) {
		t.Fatalf("parseWikilinks = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("link[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestSyncNoteLinksRebuild covers the live-save path: saving a note replaces its outgoing links from
// the current content, deduping, skipping self-links and unresolved titles, and dropping stale links
// on a later edit.
func TestSyncNoteLinksRebuild(t *testing.T) {
	dbq, ctx := newTestQueries(t)
	mk := func(title string) int64 {
		n, err := dbq.CreateNote(ctx, db.CreateNoteParams{Title: title, Content: ""})
		if err != nil {
			t.Fatalf("create %q: %v", title, err)
		}
		return n.ID
	}
	alpha, beta, gamma := mk("Alpha"), mk("Beta"), mk("Gamma")

	// Alpha → Beta (twice → dedup), Gamma, itself (skip), and a missing title (skip).
	syncNoteLinks(ctx, dbq, alpha, "[[Beta]] [[Gamma]] [[Beta]] [[Alpha]] [[Missing]]")

	betaBL, _ := dbq.GetBacklinks(ctx, beta)
	gammaBL, _ := dbq.GetBacklinks(ctx, gamma)
	alphaBL, _ := dbq.GetBacklinks(ctx, alpha)
	if len(betaBL) != 1 || betaBL[0].ID != alpha {
		t.Errorf("Beta backlinks = %v, want [Alpha]", betaBL)
	}
	if len(gammaBL) != 1 || gammaBL[0].ID != alpha {
		t.Errorf("Gamma backlinks = %v, want [Alpha]", gammaBL)
	}
	if len(alphaBL) != 0 {
		t.Errorf("Alpha must not link to itself, backlinks = %v", alphaBL)
	}

	// Edit Alpha to link only to Gamma: Beta's backlink must disappear (stale links rebuilt away).
	syncNoteLinks(ctx, dbq, alpha, "only [[Gamma]] now")
	betaBL, _ = dbq.GetBacklinks(ctx, beta)
	gammaBL, _ = dbq.GetBacklinks(ctx, gamma)
	if len(betaBL) != 0 {
		t.Errorf("after edit Beta backlinks = %v, want none", betaBL)
	}
	if len(gammaBL) != 1 || gammaBL[0].ID != alpha {
		t.Errorf("after edit Gamma backlinks = %v, want [Alpha]", gammaBL)
	}
}

func TestDailyScheme(t *testing.T) {
	const prefix = "Daily Note "
	title := dailyTitle()
	if !strings.HasPrefix(title, prefix) {
		t.Fatalf("dailyTitle = %q, want %q prefix", title, prefix)
	}
	date := strings.TrimPrefix(title, prefix)
	if _, err := time.Parse("2006-01-02", date); err != nil {
		t.Errorf("dailyTitle date %q is not YYYY-MM-DD: %v", date, err)
	}

	heading := dailyHeading()
	if !strings.HasPrefix(heading, "# ") || !strings.HasSuffix(heading, "\n\n") {
		t.Errorf("dailyHeading = %q, want '# <date>\\n\\n'", heading)
	}
	if !strings.Contains(heading, date) {
		t.Errorf("dailyHeading %q should contain today's date %q", heading, date)
	}
}
