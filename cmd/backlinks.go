/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var backlinksCmd = &cobra.Command{
	Use:   "backlinks <id>",
	Short: "Show notes that link to this note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid note ID: %s", args[0])
		}

		ctx := context.Background()

		// Verify note exists
		_, err = database.GetNote(ctx, id)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("note #%d not found", id)
			}
			return err
		}

		notes, err := database.GetBacklinks(ctx, id)
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
			fmt.Printf("No backlinks found for note #%d.\n", id)
			return nil
		}

		fmt.Printf("Notes linking to #%d:\n\n", id)
		for _, note := range notes {
			fmt.Printf("#%-4d %-40s %s\n", note.ID, note.Title, note.CreatedAt.Time.Format("2006-01-02"))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(backlinksCmd)

	backlinksCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
