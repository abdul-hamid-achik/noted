/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type frontmatter struct {
	Title string   `yaml:"title"`
	Tags  []string `yaml:"tags"`
}

var importCmd = &cobra.Command{
	Use:   "import <path>",
	Short: "Import markdown files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		recursive, _ := cmd.Flags().GetBool("recursive")
		extraTags, _ := cmd.Flags().GetString("tags")

		var extraTagList []string
		if extraTags != "" {
			for _, t := range strings.Split(extraTags, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					extraTagList = append(extraTagList, t)
				}
			}
		}

		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		var files []string
		if info.IsDir() {
			if recursive {
				err = filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !fi.IsDir() && strings.HasSuffix(strings.ToLower(fi.Name()), ".md") {
						files = append(files, p)
					}
					return nil
				})
			} else {
				entries, err := os.ReadDir(path)
				if err != nil {
					return err
				}
				for _, e := range entries {
					if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
						files = append(files, filepath.Join(path, e.Name()))
					}
				}
			}
			if err != nil {
				return err
			}
		} else {
			files = []string{path}
		}

		if len(files) == 0 {
			fmt.Println("No markdown files found.")
			return nil
		}

		ctx := context.Background()
		imported := 0

		for _, file := range files {
			title, content, fileTags, err := parseMarkdownFile(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error parsing %s: %v\n", file, err)
				continue
			}

			// Create a new slice to avoid modifying the original
			allTags := make([]string, 0, len(fileTags)+len(extraTagList))
			allTags = append(allTags, fileTags...)
			allTags = append(allTags, extraTagList...)

			note, err := database.CreateNote(ctx, db.CreateNoteParams{
				Title:   title,
				Content: content,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "error creating note from %s: %v\n", file, err)
				continue
			}

			for _, tagName := range allTags {
				tag, err := database.CreateTag(ctx, tagName)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error creating tag %s: %v\n", tagName, err)
					continue
				}
				err = database.AddTagToNote(ctx, db.AddTagToNoteParams{
					NoteID: note.ID,
					TagID:  tag.ID,
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "error tagging note: %v\n", err)
				}
			}

			fmt.Printf("Imported #%d: %s\n", note.ID, title)
			imported++
		}

		fmt.Printf("\n%d file(s) imported.\n", imported)
		return nil
	},
}

func parseMarkdownFile(path string) (title, content string, tags []string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", nil, err
	}

	text := string(data)
	fm := frontmatter{}

	if strings.HasPrefix(text, "---\n") {
		parts := strings.SplitN(text[4:], "\n---\n", 2)
		if len(parts) == 2 {
			if err := yaml.Unmarshal([]byte(parts[0]), &fm); err == nil {
				text = strings.TrimPrefix(parts[1], "\n")
			}
		}
	}

	if fm.Title != "" {
		title = fm.Title
	} else {
		// Try to extract title from first H1
		scanner := bufio.NewScanner(strings.NewReader(text))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "# ") {
				title = strings.TrimPrefix(line, "# ")
				break
			}
		}
		if title == "" {
			// Use filename as title
			title = strings.TrimSuffix(filepath.Base(path), ".md")
		}
	}

	return title, text, fm.Tags, nil
}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().BoolP("recursive", "r", false, "Scan subdirectories")
	importCmd.Flags().StringP("tags", "T", "", "Add tags to all imported (comma-separated)")
}
