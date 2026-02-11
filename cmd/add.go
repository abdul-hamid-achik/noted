/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/spf13/cobra"
)

type addResult struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	ExpiresAt string `json:"expires_at,omitempty"`
	Source    string `json:"source,omitempty"`
	SourceRef string `json:"source_ref,omitempty"`
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new note",
	Long: `Add a new note with optional tags, TTL, and source tracking.

Examples:
  noted add -t "Meeting notes" -c "Discussed project timeline"
  noted add -t "Todo" --ttl 7d -c "Review PR by Friday"
  noted add -t "Bug" --source code-review --source-ref main.go:50`,
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		tags, _ := cmd.Flags().GetString("tags")
		content, _ := cmd.Flags().GetString("content")
		ttlStr, _ := cmd.Flags().GetString("ttl")
		source, _ := cmd.Flags().GetString("source")
		sourceRef, _ := cmd.Flags().GetString("source-ref")
		folderID, _ := cmd.Flags().GetInt64("folder")
		asJSON, _ := cmd.Flags().GetBool("json")

		if content == "" {
			// Check if stdin has piped data
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("reading stdin: %w", err)
				}
				content = string(data)
			} else {
				var err error
				content, err = openEditor()
				if err != nil {
					return err
				}
			}
		}

		// Parse TTL if provided
		var expiresAt sql.NullTime
		if ttlStr != "" {
			dur, err := parseDuration(ttlStr)
			if err != nil {
				return fmt.Errorf("invalid TTL: %w", err)
			}
			expiresAt = sql.NullTime{
				Time:  time.Now().Add(dur),
				Valid: true,
			}
		}

		// Create source values
		var sourceVal, sourceRefVal sql.NullString
		if source != "" {
			sourceVal = sql.NullString{String: source, Valid: true}
		}
		if sourceRef != "" {
			sourceRefVal = sql.NullString{String: sourceRef, Valid: true}
		}

		ctx := context.Background()
		note, err := database.CreateNoteWithTTL(ctx, db.CreateNoteWithTTLParams{
			Title:     title,
			Content:   content,
			ExpiresAt: expiresAt,
			Source:    sourceVal,
			SourceRef: sourceRefVal,
		})

		if err != nil {
			return err
		}

		// Move to folder if specified
		if cmd.Flags().Changed("folder") {
			err = database.MoveNoteToFolder(ctx, db.MoveNoteToFolderParams{
				FolderID: sql.NullInt64{Int64: folderID, Valid: true},
				ID:       note.ID,
			})
			if err != nil {
				return fmt.Errorf("failed to assign folder: %w", err)
			}
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
					TagID:  tag.ID,
				})

				if err != nil {
					return err
				}
			}
		}

		if asJSON {
			result := addResult{
				ID:    note.ID,
				Title: note.Title,
			}
			if expiresAt.Valid {
				result.ExpiresAt = expiresAt.Time.Format(time.RFC3339)
			}
			if sourceVal.Valid {
				result.Source = sourceVal.String
			}
			if sourceRefVal.Valid {
				result.SourceRef = sourceRefVal.String
			}
			return outputJSON(result)
		}

		fmt.Printf("Created note #%d: %s\n", note.ID, note.Title)
		if expiresAt.Valid {
			fmt.Printf("  Expires: %s\n", expiresAt.Time.Format("2006-01-02 15:04"))
		}
		if sourceVal.Valid {
			fmt.Printf("  Source: %s\n", sourceVal.String)
			if sourceRefVal.Valid {
				fmt.Printf("  Reference: %s\n", sourceRefVal.String)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringP("title", "t", "", "Note title (required)")
	addCmd.Flags().StringP("tags", "T", "", "Comma-separated tags")
	addCmd.Flags().StringP("content", "c", "", "Note content")
	addCmd.Flags().String("ttl", "", "Time-to-live duration (e.g., '24h', '7d')")
	addCmd.Flags().String("source", "", "Source identifier (e.g., 'code-review', 'manual')")
	addCmd.Flags().String("source-ref", "", "Source reference (e.g., 'main.go:50')")
	addCmd.Flags().Int64("folder", 0, "Folder ID to add the note to")
	addCmd.Flags().BoolP("json", "j", false, "Output as JSON")

	_ = addCmd.MarkFlagRequired("title")
}
