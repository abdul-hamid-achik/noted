package tui

import (
	"context"
	"regexp"
	"strings"

	"github.com/abdul-hamid-achik/noted/internal/db"
)

// wikilinkRe matches Obsidian-style [[wikilinks]] (same pattern as the CLI's links command).
var wikilinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// parseWikilinks returns the distinct link targets in content. [[Title|alias]] resolves to "Title".
func parseWikilinks(content string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range wikilinkRe.FindAllStringSubmatch(content, -1) {
		t := strings.TrimSpace(m[1])
		if i := strings.Index(t, "|"); i >= 0 {
			t = strings.TrimSpace(t[:i])
		}
		if t != "" && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

// syncNoteLinks rewrites the note_links rows for a source note from its [[wikilinks]] (best-effort;
// unresolved titles are skipped). Keeps backlinks and the graph accurate after a save.
func syncNoteLinks(ctx context.Context, dbq *db.Queries, sourceID int64, content string) {
	if dbq == nil {
		return
	}
	_ = dbq.DeleteNoteLinks(ctx, sourceID)
	added := map[int64]bool{}
	for _, title := range parseWikilinks(content) {
		target, err := dbq.GetNoteByTitle(ctx, title)
		if err != nil || target.ID == sourceID || added[target.ID] {
			continue
		}
		added[target.ID] = true
		_ = dbq.CreateNoteLink(ctx, db.CreateNoteLinkParams{
			SourceNoteID: sourceID, TargetNoteID: target.ID, LinkText: title,
		})
	}
}
