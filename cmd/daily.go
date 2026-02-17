/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/spf13/cobra"
)

const dailyDateFormat = "2006-01-02"
const dailyFolderName = "Daily Notes"
const dailyTagName = "daily"

type dailyNoteResult struct {
	ID        int64    `json:"id"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Tags      []string `json:"tags"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

type dailyListItem struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
}

var dailyCmd = &cobra.Command{
	Use:   "daily",
	Short: "Create or show today's daily note",
	Long: `Manage daily notes in Obsidian style. Creates a note titled with the date,
tagged "daily", and stored in a "Daily Notes" folder.

Examples:
  noted daily                              # Show/create today's daily note
  noted daily --append "- [ ] Buy milk"    # Append to today's note
  noted daily --prepend "Morning thoughts" # Prepend to today's note
  noted daily --yesterday                  # Show/create yesterday's note
  noted daily --date 2026-02-14            # Show/create note for a specific date
  noted daily --list                       # List recent daily notes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")
		listMode, _ := cmd.Flags().GetBool("list")
		yesterday, _ := cmd.Flags().GetBool("yesterday")
		dateStr, _ := cmd.Flags().GetString("date")
		appendText, _ := cmd.Flags().GetString("append")
		prependText, _ := cmd.Flags().GetString("prepend")

		ctx := context.Background()

		if listMode {
			return dailyList(ctx, asJSON)
		}

		targetDate := time.Now()
		if yesterday {
			targetDate = targetDate.AddDate(0, 0, -1)
		}
		if dateStr != "" {
			parsed, err := time.Parse(dailyDateFormat, dateStr)
			if err != nil {
				return fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", err)
			}
			targetDate = parsed
		}

		title := targetDate.Format(dailyDateFormat)

		note, err := getOrCreateDailyNote(ctx, title)
		if err != nil {
			return err
		}

		if appendText != "" {
			content := note.Content
			if content != "" && !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			content += appendText
			note, err = database.UpdateNote(ctx, db.UpdateNoteParams{
				Title:   note.Title,
				Content: content,
				ID:      note.ID,
			})
			if err != nil {
				return fmt.Errorf("failed to append to daily note: %w", err)
			}
		}

		if prependText != "" {
			content := prependText
			if note.Content != "" {
				content += "\n" + note.Content
			}
			note, err = database.UpdateNote(ctx, db.UpdateNoteParams{
				Title:   note.Title,
				Content: content,
				ID:      note.ID,
			})
			if err != nil {
				return fmt.Errorf("failed to prepend to daily note: %w", err)
			}
		}

		return displayDailyNote(ctx, note, asJSON)
	},
}

func getOrCreateDailyNote(ctx context.Context, title string) (db.Note, error) {
	note, err := database.GetNoteByTitle(ctx, title)
	if err == nil {
		return note, nil
	}
	if err != sql.ErrNoRows {
		return db.Note{}, fmt.Errorf("failed to look up daily note: %w", err)
	}

	// Create new daily note
	note, err = database.CreateNoteWithTTL(ctx, db.CreateNoteWithTTLParams{
		Title:   title,
		Content: "",
	})
	if err != nil {
		return db.Note{}, fmt.Errorf("failed to create daily note: %w", err)
	}

	// Tag as "daily"
	tag, err := database.CreateTag(ctx, dailyTagName)
	if err != nil {
		return db.Note{}, fmt.Errorf("failed to create daily tag: %w", err)
	}
	err = database.AddTagToNote(ctx, db.AddTagToNoteParams{
		NoteID: note.ID,
		TagID:  tag.ID,
	})
	if err != nil {
		return db.Note{}, fmt.Errorf("failed to tag daily note: %w", err)
	}

	// Find or create "Daily Notes" folder and move note into it
	folderID, err := getOrCreateDailyFolder(ctx)
	if err != nil {
		return db.Note{}, err
	}
	err = database.MoveNoteToFolder(ctx, db.MoveNoteToFolderParams{
		FolderID: sql.NullInt64{Int64: folderID, Valid: true},
		ID:       note.ID,
	})
	if err != nil {
		return db.Note{}, fmt.Errorf("failed to move note to Daily Notes folder: %w", err)
	}

	return note, nil
}

func getOrCreateDailyFolder(ctx context.Context) (int64, error) {
	folders, err := database.ListFolders(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list folders: %w", err)
	}
	for _, f := range folders {
		if f.Name == dailyFolderName {
			return f.ID, nil
		}
	}

	folder, err := database.CreateFolder(ctx, db.CreateFolderParams{
		Name: dailyFolderName,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create Daily Notes folder: %w", err)
	}
	return folder.ID, nil
}

func displayDailyNote(ctx context.Context, note db.Note, asJSON bool) error {
	tags, err := database.GetTagsForNote(ctx, note.ID)
	if err != nil {
		return err
	}

	tagNames := make([]string, len(tags))
	for i, t := range tags {
		tagNames[i] = t.Name
	}

	if asJSON {
		result := dailyNoteResult{
			ID:        note.ID,
			Title:     note.Title,
			Content:   note.Content,
			Tags:      tagNames,
			CreatedAt: note.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: note.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
		return outputJSON(result)
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
}

func dailyList(ctx context.Context, asJSON bool) error {
	notes, err := database.GetNotesByTagName(ctx, dailyTagName)
	if err != nil {
		return fmt.Errorf("failed to list daily notes: %w", err)
	}

	// Filter to last 30 days
	cutoff := time.Now().AddDate(0, 0, -30)
	var recent []db.Note
	for _, n := range notes {
		if n.CreatedAt.Valid && n.CreatedAt.Time.After(cutoff) {
			recent = append(recent, n)
		}
	}

	if asJSON {
		items := make([]dailyListItem, len(recent))
		for i, note := range recent {
			items[i] = dailyListItem{
				ID:        note.ID,
				Title:     note.Title,
				CreatedAt: note.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
			}
		}
		return outputJSON(items)
	}

	if len(recent) == 0 {
		fmt.Println("No daily notes in the last 30 days.")
		return nil
	}

	for _, note := range recent {
		fmt.Printf("#%-4d %-40s %s\n", note.ID, note.Title, note.CreatedAt.Time.Format("2006-01-02"))
	}

	return nil
}

func init() {
	rootCmd.AddCommand(dailyCmd)

	dailyCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	dailyCmd.Flags().BoolP("list", "l", false, "List recent daily notes (last 30 days)")
	dailyCmd.Flags().BoolP("yesterday", "y", false, "Show/create yesterday's daily note")
	dailyCmd.Flags().StringP("date", "d", "", "Show/create daily note for a specific date (YYYY-MM-DD)")
	dailyCmd.Flags().StringP("append", "a", "", "Append content to the daily note")
	dailyCmd.Flags().StringP("prepend", "p", "", "Prepend content to the daily note")
}
