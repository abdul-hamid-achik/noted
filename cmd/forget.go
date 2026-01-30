/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/abdul-hamid-achik/noted/internal/config"
	"github.com/abdul-hamid-achik/noted/internal/memory"
	"github.com/abdul-hamid-achik/noted/internal/veclite"
	"github.com/spf13/cobra"
)

type forgetResultItem struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Category   string `json:"category"`
	Importance int    `json:"importance"`
}

type forgetResultOutput struct {
	DryRun      bool               `json:"dry_run,omitempty"`
	Deleted     int                `json:"deleted,omitempty"`
	WouldDelete int                `json:"would_delete,omitempty"`
	Memories    []forgetResultItem `json:"memories"`
}

var forgetCmd = &cobra.Command{
	Use:   "forget",
	Short: "Delete old or low-importance memories",
	Long: `Delete memories based on criteria like age, importance, or category.

By default, runs in dry-run mode to show what would be deleted.
Use --force to actually delete the memories.

Examples:
  noted forget --older-than 30d              # Preview memories older than 30 days
  noted forget --older-than 30d --force      # Actually delete them
  noted forget --importance-below 2          # Delete low-importance memories
  noted forget --category todo --older-than 7d
  noted forget --query "temporary"           # Delete memories matching query
  noted forget --id 42 --force               # Delete specific memory by ID`,
	RunE: func(cmd *cobra.Command, args []string) error {
		olderThan, _ := cmd.Flags().GetString("older-than")
		importanceBelow, _ := cmd.Flags().GetInt("importance-below")
		category, _ := cmd.Flags().GetString("category")
		query, _ := cmd.Flags().GetString("query")
		id, _ := cmd.Flags().GetInt64("id")
		force, _ := cmd.Flags().GetBool("force")
		asJSON, _ := cmd.Flags().GetBool("json")

		// Parse older-than to days
		var olderThanDays int
		if olderThan != "" {
			dur, err := parseDuration(olderThan)
			if err != nil {
				return fmt.Errorf("invalid --older-than: %w", err)
			}
			olderThanDays = int(dur.Hours() / 24)
			if olderThanDays < 1 {
				olderThanDays = 1
			}
		}

		// Check if any criteria specified
		if olderThanDays == 0 && importanceBelow == 0 && category == "" && query == "" && id == 0 {
			return fmt.Errorf("at least one filter criteria is required (--older-than, --importance-below, --category, --query, or --id)")
		}

		// Try to get veclite syncer
		var syncer *veclite.Syncer
		cfg, err := config.Load()
		if err == nil && cfg.VeclitePath != "" {
			syncer, _ = veclite.NewSyncer(cfg.VeclitePath, cfg.EmbeddingModel)
			if syncer != nil {
				defer syncer.Close()
			}
		}

		ctx := context.Background()

		// First, do a dry run to see what would be deleted
		result, err := memory.Forget(ctx, database, syncer, memory.ForgetInput{
			OlderThanDays:   olderThanDays,
			ImportanceBelow: importanceBelow,
			Category:        category,
			Query:           query,
			ID:              id,
			DryRun:          true, // Always dry run first
		})
		if err != nil {
			return err
		}

		if len(result.Memories) == 0 {
			if asJSON {
				return outputJSON(forgetResultOutput{
					DryRun:   !force,
					Deleted:  0,
					Memories: []forgetResultItem{},
				})
			}
			fmt.Println("No memories match the specified criteria.")
			return nil
		}

		// Show what would be deleted
		if !force || !asJSON {
			if !asJSON {
				if force {
					fmt.Printf("The following %d memories will be deleted:\n\n", len(result.Memories))
				} else {
					fmt.Printf("The following %d memories would be deleted (dry run):\n\n", len(result.Memories))
				}

				for _, mem := range result.Memories {
					fmt.Printf("#%-4d [%s] (%d) %s\n", mem.ID, mem.Category, mem.Importance, mem.Title)
				}
				fmt.Println()
			}
		}

		// If not forcing, return dry run result
		if !force {
			if asJSON {
				output := forgetResultOutput{
					DryRun:      true,
					WouldDelete: len(result.Memories),
					Memories:    make([]forgetResultItem, len(result.Memories)),
				}
				for i, mem := range result.Memories {
					output.Memories[i] = forgetResultItem{
						ID:         mem.ID,
						Title:      mem.Title,
						Category:   mem.Category,
						Importance: mem.Importance,
					}
				}
				return outputJSON(output)
			}
			fmt.Println("Use --force to actually delete these memories.")
			return nil
		}

		// Confirm deletion unless JSON mode
		if !asJSON {
			fmt.Printf("Are you sure you want to delete %d memories? [y/N]: ", len(result.Memories))
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		// Actually delete
		deleteResult, err := memory.Forget(ctx, database, syncer, memory.ForgetInput{
			OlderThanDays:   olderThanDays,
			ImportanceBelow: importanceBelow,
			Category:        category,
			Query:           query,
			ID:              id,
			DryRun:          false,
		})
		if err != nil {
			return err
		}

		if asJSON {
			output := forgetResultOutput{
				DryRun:   false,
				Deleted:  deleteResult.Deleted,
				Memories: make([]forgetResultItem, len(deleteResult.Memories)),
			}
			for i, mem := range deleteResult.Memories {
				output.Memories[i] = forgetResultItem{
					ID:         mem.ID,
					Title:      mem.Title,
					Category:   mem.Category,
					Importance: mem.Importance,
				}
			}
			return outputJSON(output)
		}

		fmt.Printf("Deleted %d memories.\n", deleteResult.Deleted)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(forgetCmd)

	forgetCmd.Flags().String("older-than", "", "Delete memories older than duration (e.g., '30d', '24h')")
	forgetCmd.Flags().Int("importance-below", 0, "Delete memories below this importance level (1-5)")
	forgetCmd.Flags().StringP("category", "c", "", "Only delete memories in this category")
	forgetCmd.Flags().StringP("query", "q", "", "Delete memories matching this query")
	forgetCmd.Flags().Int64("id", 0, "Delete specific memory by ID")
	forgetCmd.Flags().BoolP("force", "f", false, "Actually delete (without this flag, only shows what would be deleted)")
	forgetCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
