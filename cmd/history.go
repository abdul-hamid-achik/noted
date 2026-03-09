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

type versionListItem struct {
	NoteID        int64  `json:"note_id"`
	VersionNumber int64  `json:"version_number"`
	Title         string `json:"title"`
	CreatedAt     string `json:"created_at"`
}

type versionDetail struct {
	NoteID        int64  `json:"note_id"`
	VersionNumber int64  `json:"version_number"`
	Title         string `json:"title"`
	Content       string `json:"content"`
	CreatedAt     string `json:"created_at"`
}

var historyCmd = &cobra.Command{
	Use:   "history <id>",
	Short: "List version history of a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")
		versionNum, _ := cmd.Flags().GetInt("version")

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid note ID: %s", args[0])
		}

		ctx := context.Background()

		// Verify note exists
		note, err := database.GetNote(ctx, id)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("note #%d not found", id)
			}
			return fmt.Errorf("failed to get note: %w", err)
		}

		// If --version is specified, show that specific version
		if cmd.Flags().Changed("version") {
			version, err := database.GetNoteVersion(ctx, db.GetNoteVersionParams{
				NoteID:        id,
				VersionNumber: int64(versionNum),
			})
			if err != nil {
				if err == sql.ErrNoRows {
					return fmt.Errorf("version %d not found for note #%d", versionNum, id)
				}
				return fmt.Errorf("failed to get version: %w", err)
			}

			if asJSON {
				return outputJSON(versionDetail{
					NoteID:        id,
					VersionNumber: version.VersionNumber,
					Title:         version.Title,
					Content:       version.Content,
					CreatedAt:     version.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
				})
			}

			fmt.Printf("# %s (version %d)\n\n", version.Title, version.VersionNumber)
			fmt.Printf("Created: %s\n", version.CreatedAt.Time.Format("2006-01-02 15:04"))
			fmt.Printf("\n---\n\n%s", version.Content)
			if len(version.Content) > 0 && version.Content[len(version.Content)-1] != '\n' {
				fmt.Println()
			}
			return nil
		}

		// List all versions
		versions, err := database.GetNoteVersions(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get versions: %w", err)
		}

		if len(versions) == 0 {
			fmt.Printf("No version history for note #%d (%s)\n", id, note.Title)
			return nil
		}

		if asJSON {
			items := make([]versionListItem, len(versions))
			for i, v := range versions {
				items[i] = versionListItem{
					NoteID:        id,
					VersionNumber: v.VersionNumber,
					Title:         v.Title,
					CreatedAt:     v.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
				}
			}
			return outputJSON(items)
		}

		fmt.Printf("Version history for note #%d (%s):\n\n", id, note.Title)
		for _, v := range versions {
			fmt.Printf("  v%-4d  %s  %s\n", v.VersionNumber, v.CreatedAt.Time.Format("2006-01-02 15:04"), v.Title)
		}
		return nil
	},
}

var diffCmd = &cobra.Command{
	Use:   "diff <id>",
	Short: "Show diff between a version and current note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")
		versionNum, _ := cmd.Flags().GetInt("version")

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

		// Determine which version to diff against
		var version db.NoteVersion
		if cmd.Flags().Changed("version") {
			version, err = database.GetNoteVersion(ctx, db.GetNoteVersionParams{
				NoteID:        id,
				VersionNumber: int64(versionNum),
			})
			if err != nil {
				if err == sql.ErrNoRows {
					return fmt.Errorf("version %d not found for note #%d", versionNum, id)
				}
				return fmt.Errorf("failed to get version: %w", err)
			}
		} else {
			// Use latest version
			versions, err := database.GetNoteVersions(ctx, id)
			if err != nil {
				return fmt.Errorf("failed to get versions: %w", err)
			}
			if len(versions) == 0 {
				fmt.Println("No version history to diff against.")
				return nil
			}
			version = versions[0] // Already sorted DESC
		}

		diff := lineDiff(version.Content, note.Content)

		if asJSON {
			return outputJSON(struct {
				NoteID        int64  `json:"note_id"`
				VersionNumber int64  `json:"version_number"`
				Diff          string `json:"diff"`
			}{
				NoteID:        id,
				VersionNumber: version.VersionNumber,
				Diff:          diff,
			})
		}

		fmt.Printf("Diff: note #%d version %d -> current\n\n", id, version.VersionNumber)
		fmt.Print(diff)
		return nil
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore <id>",
	Short: "Restore a note to a previous version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")
		versionNum, _ := cmd.Flags().GetInt("version")

		if !cmd.Flags().Changed("version") {
			return fmt.Errorf("--version flag is required")
		}

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid note ID: %s", args[0])
		}

		ctx := context.Background()

		// Get current note
		note, err := database.GetNote(ctx, id)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("note #%d not found", id)
			}
			return fmt.Errorf("failed to get note: %w", err)
		}

		// Get the version to restore
		version, err := database.GetNoteVersion(ctx, db.GetNoteVersionParams{
			NoteID:        id,
			VersionNumber: int64(versionNum),
		})
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("version %d not found for note #%d", versionNum, id)
			}
			return fmt.Errorf("failed to get version: %w", err)
		}

		// Save current state as a new version before restoring
		latestVerNum, err := getLatestVersionNumber(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get latest version number: %w", err)
		}

		_, err = database.CreateNoteVersion(ctx, db.CreateNoteVersionParams{
			NoteID:        id,
			Title:         note.Title,
			Content:       note.Content,
			VersionNumber: latestVerNum + 1,
		})
		if err != nil {
			return fmt.Errorf("failed to save current state: %w", err)
		}

		// Restore the note to the target version
		_, err = database.UpdateNote(ctx, db.UpdateNoteParams{
			ID:      id,
			Title:   version.Title,
			Content: version.Content,
		})
		if err != nil {
			return fmt.Errorf("failed to restore note: %w", err)
		}

		if asJSON {
			return outputJSON(struct {
				ID              int64  `json:"id"`
				RestoredVersion int64  `json:"restored_version"`
				Title           string `json:"title"`
			}{
				ID:              id,
				RestoredVersion: version.VersionNumber,
				Title:           version.Title,
			})
		}

		fmt.Printf("Restored note #%d to version %d\n", id, versionNum)
		return nil
	},
}

// getLatestVersionNumber retrieves the latest version number for a note.
func getLatestVersionNumber(ctx context.Context, noteID int64) (int64, error) {
	result, err := database.GetLatestVersionNumber(ctx, noteID)
	if err != nil {
		return 0, err
	}
	// COALESCE returns interface{} via sqlc; handle the type assertion
	switch v := result.(type) {
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("unexpected type from GetLatestVersionNumber: %T", result)
	}
}

// lineDiff produces a simple line-by-line diff between old and new content.
func lineDiff(old, new string) string {
	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	var b strings.Builder

	// Use a simple LCS-based approach for reasonable diffs
	i, j := 0, 0
	for i < len(oldLines) || j < len(newLines) {
		if i < len(oldLines) && j < len(newLines) && oldLines[i] == newLines[j] {
			b.WriteString("  " + oldLines[i] + "\n")
			i++
			j++
		} else {
			// Look ahead to find next matching line
			foundOld, foundNew := -1, -1

			// Search for old[i] in upcoming new lines
			for k := j; k < len(newLines) && k < j+10; k++ {
				if i < len(oldLines) && oldLines[i] == newLines[k] {
					foundNew = k
					break
				}
			}

			// Search for new[j] in upcoming old lines
			for k := i; k < len(oldLines) && k < i+10; k++ {
				if j < len(newLines) && newLines[j] == oldLines[k] {
					foundOld = k
					break
				}
			}

			if foundNew >= 0 && (foundOld < 0 || (foundNew-j) <= (foundOld-i)) {
				// New lines were added
				for j < foundNew {
					b.WriteString("+ " + newLines[j] + "\n")
					j++
				}
			} else if foundOld >= 0 {
				// Old lines were removed
				for i < foundOld {
					b.WriteString("- " + oldLines[i] + "\n")
					i++
				}
			} else {
				// Lines differ
				if i < len(oldLines) {
					b.WriteString("- " + oldLines[i] + "\n")
					i++
				}
				if j < len(newLines) {
					b.WriteString("+ " + newLines[j] + "\n")
					j++
				}
			}
		}
	}

	return b.String()
}

func init() {
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(restoreCmd)

	historyCmd.Flags().IntP("version", "v", 0, "Show a specific version")
	historyCmd.Flags().BoolP("json", "j", false, "Output as JSON")

	diffCmd.Flags().IntP("version", "v", 0, "Diff against a specific version")
	diffCmd.Flags().BoolP("json", "j", false, "Output as JSON")

	restoreCmd.Flags().IntP("version", "v", 0, "Version to restore (required)")
	restoreCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
