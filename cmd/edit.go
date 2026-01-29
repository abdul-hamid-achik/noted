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

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/spf13/cobra"
)

type editResult struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		content, _ := cmd.Flags().GetString("content")
		tags, _ := cmd.Flags().GetString("tags")
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

		newTitle := note.Title
		newContent := note.Content

		if cmd.Flags().Changed("title") {
			newTitle = title
		}

		if cmd.Flags().Changed("content") {
			newContent = content
		} else if !cmd.Flags().Changed("title") && !cmd.Flags().Changed("tags") {
			// No flags provided, open editor with current content
			edited, err := openEditorWithContent(note.Content)
			if err != nil {
				return err
			}
			newContent = edited
		}

		_, err = database.UpdateNote(ctx, db.UpdateNoteParams{
			ID:      id,
			Title:   newTitle,
			Content: newContent,
		})
		if err != nil {
			return err
		}

		if cmd.Flags().Changed("tags") {
			if err := database.RemoveAllTagsFromNote(ctx, id); err != nil {
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
						NoteID: id,
						TagID:  tag.ID,
					})
					if err != nil {
						return err
					}
				}
			}
		}

		if asJSON {
			return outputJSON(editResult{
				ID:    id,
				Title: newTitle,
			})
		}

		fmt.Printf("Updated note #%d\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)

	editCmd.Flags().StringP("title", "t", "", "New title")
	editCmd.Flags().StringP("content", "c", "", "New content")
	editCmd.Flags().StringP("tags", "T", "", "Replace tags (comma-separated)")
	editCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
