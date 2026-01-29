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

var grepCmd = &cobra.Command{
	Use:   "grep <pattern>",
	Short: "Search notes by text",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pattern := args[0]
		limit, _ := cmd.Flags().GetInt("limit")

		if limit < 1 {
			return fmt.Errorf("limit must be at least 1")
		}

		// SQLite LIKE is case-insensitive for ASCII by default
		// For Unicode case-insensitivity, we'd need COLLATE NOCASE or application-level filtering
		searchPattern := "%" + pattern + "%"

		ctx := context.Background()
		notes, err := database.SearchNotesContent(ctx, db.SearchNotesContentParams{
			Content: searchPattern,
			Title:   searchPattern,
			Limit:   int64(limit),
		})
		if err != nil {
			return err
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
}
