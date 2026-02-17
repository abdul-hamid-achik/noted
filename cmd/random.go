/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/spf13/cobra"
)

var randomCmd = &cobra.Command{
	Use:   "random",
	Short: "Display a random note",
	Long: `Display a random note from your knowledge base.

Examples:
  noted random
  noted random --tag work
  noted random --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tag, _ := cmd.Flags().GetString("tag")
		asJSON, _ := cmd.Flags().GetBool("json")

		ctx := context.Background()
		var notes []db.Note
		var err error

		if tag != "" {
			notes, err = database.GetNotesByTagName(ctx, tag)
		} else {
			notes, err = database.GetAllNotes(ctx)
		}
		if err != nil {
			return fmt.Errorf("failed to get notes: %w", err)
		}

		if len(notes) == 0 {
			if tag != "" {
				fmt.Printf("No notes found with tag %q.\n", tag)
			} else {
				fmt.Println("No notes found.")
			}
			return nil
		}

		note := notes[rand.IntN(len(notes))]

		tags, err := database.GetTagsForNote(ctx, note.ID)
		if err != nil {
			return err
		}

		tagNames := make([]string, len(tags))
		for i, t := range tags {
			tagNames[i] = t.Name
		}

		if asJSON {
			detail := noteDetail{
				ID:        note.ID,
				Title:     note.Title,
				Content:   note.Content,
				Tags:      tagNames,
				CreatedAt: note.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt: note.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
			}
			return outputJSON(detail)
		}

		fmt.Printf("# %s\n\n", note.Title)
		fmt.Printf("ID: %d\n", note.ID)
		fmt.Printf("Created: %s\n", note.CreatedAt.Time.Format("2006-01-02 15:04"))
		fmt.Printf("Updated: %s\n", note.UpdatedAt.Time.Format("2006-01-02 15:04"))

		if len(tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(tagNames, ", "))
		}

		fmt.Printf("\n---\n\n%s", note.Content)
		if len(note.Content) > 0 && note.Content[len(note.Content)-1] != '\n' {
			fmt.Println()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(randomCmd)

	randomCmd.Flags().StringP("tag", "T", "", "Pick from notes with this tag")
	randomCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
