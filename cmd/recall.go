/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/abdul-hamid-achik/noted/internal/config"
	"github.com/abdul-hamid-achik/noted/internal/memory"
	"github.com/abdul-hamid-achik/noted/internal/veclite"
	"github.com/spf13/cobra"
)

type recallResultItem struct {
	ID         int64   `json:"id"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Category   string  `json:"category"`
	Importance int     `json:"importance"`
	Score      float64 `json:"score,omitempty"`
	Source     string  `json:"source,omitempty"`
	SourceRef  string  `json:"source_ref,omitempty"`
}

type recallResultOutput struct {
	Query    string             `json:"query"`
	Method   string             `json:"method"`
	Count    int                `json:"count"`
	Memories []recallResultItem `json:"memories"`
}

var recallCmd = &cobra.Command{
	Use:   "recall <query>",
	Short: "Recall memories by search query",
	Long: `Search for relevant memories using semantic or keyword search.

Uses semantic search if available (requires Ollama), otherwise falls back
to keyword search.

Examples:
  noted recall "database conventions"
  noted recall "authentication" --limit 10
  noted recall "project setup" --category project
  noted recall "JWT" --semantic`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		limit, _ := cmd.Flags().GetInt("limit")
		category, _ := cmd.Flags().GetString("category")
		semantic, _ := cmd.Flags().GetBool("semantic")
		asJSON, _ := cmd.Flags().GetBool("json")

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
		result, err := memory.Recall(ctx, database, syncer, memory.RecallInput{
			Query:       query,
			Limit:       limit,
			Category:    category,
			UseSemantic: semantic && syncer != nil,
		})
		if err != nil {
			return err
		}

		if asJSON {
			output := recallResultOutput{
				Query:    result.Query,
				Method:   result.Method,
				Count:    result.Count,
				Memories: make([]recallResultItem, len(result.Memories)),
			}
			for i, mem := range result.Memories {
				output.Memories[i] = recallResultItem{
					ID:         mem.ID,
					Title:      mem.Title,
					Content:    mem.Content,
					Category:   mem.Category,
					Importance: mem.Importance,
					Score:      mem.Score,
					Source:     mem.Source,
					SourceRef:  mem.SourceRef,
				}
			}
			return outputJSON(output)
		}

		if result.Count == 0 {
			fmt.Println("No memories found.")
			return nil
		}

		fmt.Printf("Found %d memories (via %s search):\n\n", result.Count, result.Method)
		for _, mem := range result.Memories {
			fmt.Printf("#%-4d [%s] %s\n", mem.ID, mem.Category, mem.Title)

			// Show score if available
			if mem.Score > 0 {
				fmt.Printf("      Score: %.2f\n", mem.Score)
			}

			// Show importance
			importanceStars := ""
			for i := 0; i < mem.Importance; i++ {
				importanceStars += "*"
			}
			fmt.Printf("      Importance: %s (%d/5)\n", importanceStars, mem.Importance)

			// Show content preview
			content := mem.Content
			if len(content) > 100 {
				content = content[:100] + "..."
			}
			fmt.Printf("      %s\n", content)

			// Show source if available
			if mem.Source != "" {
				if mem.SourceRef != "" {
					fmt.Printf("      Source: %s @ %s\n", mem.Source, mem.SourceRef)
				} else {
					fmt.Printf("      Source: %s\n", mem.Source)
				}
			}

			// Show expiration if set
			if !mem.ExpiresAt.IsZero() {
				if mem.ExpiresAt.Before(time.Now()) {
					fmt.Printf("      [EXPIRED]\n")
				} else {
					fmt.Printf("      Expires: %s\n", mem.ExpiresAt.Format("2006-01-02 15:04"))
				}
			}

			fmt.Println()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(recallCmd)

	recallCmd.Flags().IntP("limit", "n", 5, "Max results to return")
	recallCmd.Flags().StringP("category", "c", "", "Filter by category")
	recallCmd.Flags().BoolP("semantic", "s", true, "Use semantic search if available")
	recallCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
