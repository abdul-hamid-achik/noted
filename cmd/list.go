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

type noteListItem struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all notes",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, err := cmd.Flags().GetInt("limit")
		if err != nil {
			return err
		}

		tag, err := cmd.Flags().GetString("tag")
		if err != nil {
			return err
		}

		asJSON, _ := cmd.Flags().GetBool("json")

		ctx := context.Background()
		var notes []db.Note

		if tag != "" {
			notes, err = database.GetNotesByTagName(ctx, tag)
		} else {
			notes, err = database.ListNotes(ctx, db.ListNotesParams{
				Limit:  int64(limit),
				Offset: 0,
			})
		}
		if err != nil {
			return err
		}

		if asJSON {
			items := make([]noteListItem, len(notes))
			for i, note := range notes {
				items[i] = noteListItem{
					ID:        note.ID,
					Title:     note.Title,
					CreatedAt: note.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
				}
			}
			return outputJSON(items)
		}

		if len(notes) == 0 {
			fmt.Println("No notes found.")
			return nil
		}

		for _, note := range notes {
			fmt.Printf("#%-4d %-40s %s\n", note.ID, note.Title, note.CreatedAt.Time.Format("2006-01-02"))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().IntP("limit", "n", 20, "Max number of notes to show")
	listCmd.Flags().StringP("tag", "T", "", "Filter by tag name")
	listCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
