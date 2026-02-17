/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
)

var wikilinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

type orphanItem struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type unresolvedItem struct {
	LinkText   string `json:"link_text"`
	SourceID   int64  `json:"source_id"`
	SourceNote string `json:"source_note"`
}

var orphansCmd = &cobra.Command{
	Use:   "orphans",
	Short: "Find notes with no links in or out",
	Long: `Find orphan notes — notes that have no incoming and no outgoing links.
These are isolated nodes in your knowledge graph.

Examples:
  noted orphans
  noted orphans --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")
		ctx := context.Background()

		rows, err := conn.QueryContext(ctx, `
			SELECT n.id, n.title FROM notes n
			WHERE n.id NOT IN (SELECT source_note_id FROM note_links)
			AND n.id NOT IN (SELECT target_note_id FROM note_links)
			ORDER BY n.title`)
		if err != nil {
			return fmt.Errorf("failed to query orphan notes: %w", err)
		}
		defer rows.Close()

		var items []orphanItem
		for rows.Next() {
			var item orphanItem
			if err := rows.Scan(&item.ID, &item.Title); err != nil {
				return err
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		if asJSON {
			return outputJSON(items)
		}

		if len(items) == 0 {
			fmt.Println("No orphan notes found.")
			return nil
		}

		fmt.Println("Orphan notes (no links in or out):")
		for _, item := range items {
			fmt.Printf("  #%-4d %s\n", item.ID, item.Title)
		}
		fmt.Printf("\n%d orphan note(s) found\n", len(items))

		return nil
	},
}

var deadendsCmd = &cobra.Command{
	Use:   "deadends",
	Short: "Find notes with incoming links but no outgoing links",
	Long: `Find dead-end notes — notes that are linked to by other notes
but don't link out to anything. These are knowledge sinks.

Examples:
  noted deadends
  noted deadends --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")
		ctx := context.Background()

		rows, err := conn.QueryContext(ctx, `
			SELECT n.id, n.title FROM notes n
			WHERE n.id IN (SELECT target_note_id FROM note_links)
			AND n.id NOT IN (SELECT source_note_id FROM note_links)
			ORDER BY n.title`)
		if err != nil {
			return fmt.Errorf("failed to query dead-end notes: %w", err)
		}
		defer rows.Close()

		var items []orphanItem
		for rows.Next() {
			var item orphanItem
			if err := rows.Scan(&item.ID, &item.Title); err != nil {
				return err
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		if asJSON {
			return outputJSON(items)
		}

		if len(items) == 0 {
			fmt.Println("No dead-end notes found.")
			return nil
		}

		fmt.Println("Dead-end notes (incoming links, no outgoing):")
		for _, item := range items {
			fmt.Printf("  #%-4d %s\n", item.ID, item.Title)
		}
		fmt.Printf("\n%d dead-end note(s) found\n", len(items))

		return nil
	},
}

var unresolvedCmd = &cobra.Command{
	Use:   "unresolved",
	Short: "Find broken wikilinks",
	Long: `Find unresolved wikilinks — [[references]] in note content
where no matching note title exists.

Examples:
  noted unresolved
  noted unresolved --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")
		ctx := context.Background()

		notes, err := database.GetAllNotes(ctx)
		if err != nil {
			return fmt.Errorf("failed to get notes: %w", err)
		}

		var items []unresolvedItem
		for _, note := range notes {
			matches := wikilinkRe.FindAllStringSubmatch(note.Content, -1)
			for _, match := range matches {
				linkText := match[1]
				_, err := database.GetNoteByTitle(ctx, linkText)
				if err == sql.ErrNoRows {
					items = append(items, unresolvedItem{
						LinkText:   linkText,
						SourceID:   note.ID,
						SourceNote: note.Title,
					})
				} else if err != nil {
					return fmt.Errorf("failed to look up note %q: %w", linkText, err)
				}
			}
		}

		if asJSON {
			return outputJSON(items)
		}

		if len(items) == 0 {
			fmt.Println("No unresolved links found.")
			return nil
		}

		fmt.Println("Unresolved links:")
		for _, item := range items {
			fmt.Printf("  [[%s]] referenced in %q (#%d)\n", item.LinkText, item.SourceNote, item.SourceID)
		}
		fmt.Printf("\n%d unresolved link(s) found\n", len(items))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(orphansCmd)
	rootCmd.AddCommand(deadendsCmd)
	rootCmd.AddCommand(unresolvedCmd)

	orphansCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	deadendsCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	unresolvedCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
