/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "A brief description of your command",
	Long: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		tags, _ := cmd.Flags().GetString("tags")
		content, _ := cmd.Flags().GetString("content")

		if content == "" {
			var err error
			content, err = openEditor()
			if err != nil {
				return err
			}
		}

		ctx := context.Background()
		note, err := database.CreateNote(ctx, db.CreateNoteParams{
			Title: title,
			Content: content,
		})

		if err != nil {
			return err
		}

		if tags != "" {
			tagList := strings.Split(tags, ",")
			for _, tagName := range tagList {
				tagName = strings.TrimSpace(tagName)
				if tagName == "" {
					continue
				}

				tag, err := database.CreateTag(ctx, tagName)
				if err != nil {
					return err
				}

				err = database.AddTagToNote(ctx, db.AddTagToNoteParams{
					NoteID: note.ID,
					TagID: tag.ID,
				})

				if err != nil {
					return err
				}
			}
		}

		fmt.Printf("Created note #%d: %s\n", note.ID, note.Title)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringP("title", "t", "", "Note title (required)")
	addCmd.Flags().StringP("tags", "T", "", "Comma-separated tags")
	addCmd.Flags().StringP("content", "c", "", "Note content")

	_ = addCmd.MarkFlagRequired("title")
}
