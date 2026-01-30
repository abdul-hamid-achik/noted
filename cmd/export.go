/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/spf13/cobra"
)

type exportedNote struct {
	ID        int64    `json:"id"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Tags      []string `json:"tags"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
	ExpiresAt string   `json:"expires_at,omitempty"`
	Source    string   `json:"source,omitempty"`
	SourceRef string   `json:"source_ref,omitempty"`
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export notes",
	Long: `Export notes to various formats.

Supported formats:
  - markdown: Single file with YAML frontmatter (default)
  - json: JSON array
  - jsonl: JSON Lines (one JSON object per line)

Examples:
  noted export                              # Export all as markdown to stdout
  noted export --format json -o notes.json  # Export as JSON to file
  noted export --format jsonl               # Export as JSON Lines
  noted export --tag project                # Export only notes with 'project' tag
  noted export --since 2025-01-01           # Export notes created since date`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")
		tag, _ := cmd.Flags().GetString("tag")
		since, _ := cmd.Flags().GetString("since")

		ctx := context.Background()
		var notes []db.Note
		var err error

		if tag != "" {
			notes, err = database.GetNotesByTagName(ctx, tag)
		} else if since != "" {
			// Parse since date
			sinceTime, parseErr := time.Parse("2006-01-02", since)
			if parseErr != nil {
				return fmt.Errorf("invalid --since date format (use YYYY-MM-DD): %w", parseErr)
			}
			notes, err = database.GetNotesSince(ctx, sql.NullTime{Time: sinceTime, Valid: true})
		} else {
			notes, err = database.GetAllNotes(ctx)
		}
		if err != nil {
			return err
		}

		if len(notes) == 0 {
			fmt.Fprintln(os.Stderr, "No notes to export.")
			return nil
		}

		var w io.Writer = os.Stdout
		if output != "" {
			f, err := os.Create(output)
			if err != nil {
				return err
			}
			defer f.Close()
			w = f
		}

		switch format {
		case "json":
			return exportJSON(ctx, w, notes)
		case "jsonl":
			return exportJSONL(ctx, w, notes)
		case "markdown":
			return exportMarkdown(ctx, w, notes)
		default:
			return fmt.Errorf("unknown format: %s (use 'markdown', 'json', or 'jsonl')", format)
		}
	},
}

func noteToExported(ctx context.Context, note db.Note) (exportedNote, error) {
	tags, err := database.GetTagsForNote(ctx, note.ID)
	if err != nil {
		return exportedNote{}, err
	}

	tagNames := make([]string, len(tags))
	for i, t := range tags {
		tagNames[i] = t.Name
	}

	exported := exportedNote{
		ID:        note.ID,
		Title:     note.Title,
		Content:   note.Content,
		Tags:      tagNames,
		CreatedAt: note.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: note.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}

	if note.ExpiresAt.Valid {
		exported.ExpiresAt = note.ExpiresAt.Time.Format("2006-01-02T15:04:05Z")
	}
	if note.Source.Valid {
		exported.Source = note.Source.String
	}
	if note.SourceRef.Valid {
		exported.SourceRef = note.SourceRef.String
	}

	return exported, nil
}

func exportJSON(ctx context.Context, w io.Writer, notes []db.Note) error {
	exported := make([]exportedNote, 0, len(notes))

	for _, note := range notes {
		exp, err := noteToExported(ctx, note)
		if err != nil {
			return err
		}
		exported = append(exported, exp)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(exported)
}

func exportJSONL(ctx context.Context, w io.Writer, notes []db.Note) error {
	encoder := json.NewEncoder(w)
	for _, note := range notes {
		exp, err := noteToExported(ctx, note)
		if err != nil {
			return err
		}
		if err := encoder.Encode(exp); err != nil {
			return err
		}
	}
	return nil
}

func exportMarkdown(ctx context.Context, w io.Writer, notes []db.Note) error {
	for i, note := range notes {
		tags, err := database.GetTagsForNote(ctx, note.ID)
		if err != nil {
			return err
		}

		tagNames := make([]string, len(tags))
		for j, t := range tags {
			tagNames[j] = t.Name
		}

		// YAML frontmatter
		fmt.Fprintln(w, "---")
		fmt.Fprintf(w, "title: %q\n", note.Title)
		if len(tagNames) > 0 {
			// Quote each tag to handle special characters
			quotedTags := make([]string, len(tagNames))
			for i, tag := range tagNames {
				quotedTags[i] = fmt.Sprintf("%q", tag)
			}
			fmt.Fprintf(w, "tags: [%s]\n", strings.Join(quotedTags, ", "))
		}
		fmt.Fprintf(w, "created: %s\n", note.CreatedAt.Time.Format("2006-01-02T15:04:05Z"))
		fmt.Fprintf(w, "updated: %s\n", note.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"))
		fmt.Fprintln(w, "---")
		fmt.Fprintln(w)
		fmt.Fprint(w, note.Content)
		if len(note.Content) > 0 && note.Content[len(note.Content)-1] != '\n' {
			fmt.Fprintln(w)
		}

		if i < len(notes)-1 {
			fmt.Fprintln(w)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringP("format", "f", "markdown", "Output format (markdown, json, jsonl)")
	exportCmd.Flags().StringP("output", "o", "", "Output path (default: stdout)")
	exportCmd.Flags().StringP("tag", "T", "", "Filter by tag")
	exportCmd.Flags().String("since", "", "Export notes created since date (YYYY-MM-DD)")
}
