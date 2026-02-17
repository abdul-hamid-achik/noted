/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/spf13/cobra"
)

type extractedTask struct {
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
	NoteID    int64  `json:"note_id"`
	NoteTitle string `json:"note_title"`
	Line      int    `json:"line"`
}

var taskRegex = regexp.MustCompile(`^\s*-\s*\[([ xX])\]\s*(.+)$`)

func extractTasks(note db.Note) []extractedTask {
	var tasks []extractedTask
	lines := strings.Split(note.Content, "\n")
	for i, line := range lines {
		matches := taskRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		tasks = append(tasks, extractedTask{
			Text:      strings.TrimSpace(matches[2]),
			Completed: matches[1] != " ",
			NoteID:    note.ID,
			NoteTitle: note.Title,
			Line:      i + 1,
		})
	}
	return tasks
}

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "List tasks extracted from notes",
	Long: `Extract and display tasks (markdown checkboxes) from your notes.

Tasks are parsed from markdown checkbox syntax:
  - [ ] pending task
  - [x] completed task

Examples:
  noted tasks
  noted tasks --pending
  noted tasks --completed
  noted tasks --tag work
  noted tasks --note 42
  noted tasks --count
  noted tasks --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pending, _ := cmd.Flags().GetBool("pending")
		completed, _ := cmd.Flags().GetBool("completed")
		tag, _ := cmd.Flags().GetString("tag")
		noteID, _ := cmd.Flags().GetInt64("note")
		countOnly, _ := cmd.Flags().GetBool("count")
		asJSON, _ := cmd.Flags().GetBool("json")

		ctx := context.Background()
		var notes []db.Note
		var err error

		if cmd.Flags().Changed("note") {
			note, err := database.GetNote(ctx, noteID)
			if err != nil {
				return err
			}
			notes = []db.Note{note}
		} else if tag != "" {
			notes, err = database.GetNotesByTagName(ctx, tag)
		} else {
			notes, err = database.GetAllNotes(ctx)
		}
		if err != nil {
			return err
		}

		var allTasks []extractedTask
		for _, note := range notes {
			allTasks = append(allTasks, extractTasks(note)...)
		}

		// Filter by status
		if pending || completed {
			var filtered []extractedTask
			for _, t := range allTasks {
				if (pending && !t.Completed) || (completed && t.Completed) {
					filtered = append(filtered, t)
				}
			}
			allTasks = filtered
		}

		// Count mode
		if countOnly {
			pendingCount := 0
			completedCount := 0
			for _, t := range allTasks {
				if t.Completed {
					completedCount++
				} else {
					pendingCount++
				}
			}
			if asJSON {
				return outputJSON(map[string]int{
					"pending":   pendingCount,
					"completed": completedCount,
					"total":     pendingCount + completedCount,
				})
			}
			fmt.Printf("Tasks: %d pending, %d completed, %d total\n", pendingCount, completedCount, pendingCount+completedCount)
			return nil
		}

		if asJSON {
			if allTasks == nil {
				allTasks = []extractedTask{}
			}
			return outputJSON(allTasks)
		}

		if len(allTasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		pendingCount := 0
		completedCount := 0
		for _, t := range allTasks {
			check := " "
			if t.Completed {
				check = "x"
				completedCount++
			} else {
				pendingCount++
			}
			fmt.Printf("[%s] %-45s (%s #%d)\n", check, t.Text, t.NoteTitle, t.NoteID)
		}
		fmt.Printf("\nTasks: %d pending, %d completed, %d total\n", pendingCount, completedCount, pendingCount+completedCount)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tasksCmd)

	tasksCmd.Flags().Bool("pending", false, "Show only pending tasks")
	tasksCmd.Flags().Bool("completed", false, "Show only completed tasks")
	tasksCmd.Flags().StringP("tag", "T", "", "Filter by tag name")
	tasksCmd.Flags().Int64("note", 0, "Filter by note ID")
	tasksCmd.Flags().Bool("count", false, "Show task counts only")
	tasksCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
