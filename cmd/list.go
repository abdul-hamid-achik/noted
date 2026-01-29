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

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "A brief description of your command",
	Long: "",
	RunE: func(cmd *cobra.Command, args []string) error {
    limit, err := cmd.Flags().GetInt("limit")
    if err != nil {
        return err
    }

    tag, err := cmd.Flags().GetString("tag")
    if err != nil {
        return err
    }

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
}
