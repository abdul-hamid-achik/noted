/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type noteDetail struct {
	ID        int64    `json:"id"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Tags      []string `json:"tags"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Display a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		raw, _ := cmd.Flags().GetBool("raw")
		asJSON, _ := cmd.Flags().GetBool("json")

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid note ID: %s", args[0])
		}

		ctx := context.Background()
		note, err := database.GetNote(ctx, id)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("note #%d not found", id)
			}
			return fmt.Errorf("failed to get note: %w", err)
		}

		if raw {
			fmt.Print(note.Content)
			return nil
		}

		tags, err := database.GetTagsForNote(ctx, id)
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
	rootCmd.AddCommand(showCmd)

	showCmd.Flags().BoolP("raw", "r", false, "Output raw markdown only")
	showCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
