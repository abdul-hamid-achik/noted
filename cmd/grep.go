/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/spf13/cobra"
)

type grepResultItem struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	UpdatedAt string `json:"updated_at"`
}

var grepCmd = &cobra.Command{
	Use:   "grep <pattern>",
	Short: "Search notes by text",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pattern := args[0]
		limit, _ := cmd.Flags().GetInt("limit")
		asJSON, _ := cmd.Flags().GetBool("json")

		if limit < 1 {
			return fmt.Errorf("limit must be at least 1")
		}

		ctx := context.Background()

		// Try FTS5 first, fall back to LIKE
		var notes []db.Note
		var err error
		if db.FTSAvailable(ctx, conn) {
			notes, err = db.SearchNotesFTS(ctx, conn, pattern, int64(limit))
		}
		if notes == nil || err != nil {
			searchPattern := "%" + pattern + "%"
			notes, err = database.SearchNotesContent(ctx, db.SearchNotesContentParams{
				Content: searchPattern,
				Title:   searchPattern,
				Limit:   int64(limit),
			})
		}
		if err != nil {
			return err
		}

		if asJSON {
			items := make([]grepResultItem, len(notes))
			for i, note := range notes {
				items[i] = grepResultItem{
					ID:        note.ID,
					Title:     note.Title,
					UpdatedAt: note.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
				}
			}
			return outputJSON(items)
		}

		if len(notes) == 0 {
			fmt.Println("No matching notes found.")
			return nil
		}

		for _, note := range notes {
			fmt.Printf("#%-4d %-40s %s\n", note.ID, note.Title, note.UpdatedAt.Time.Format("2006-01-02"))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(grepCmd)

	grepCmd.Flags().IntP("limit", "n", 20, "Max results")
	grepCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
