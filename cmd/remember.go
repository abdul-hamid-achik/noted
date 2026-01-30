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

type rememberResult struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Category   string    `json:"category"`
	Importance int       `json:"importance"`
	ExpiresAt  string    `json:"expires_at,omitempty"`
	Source     string    `json:"source,omitempty"`
	SourceRef  string    `json:"source_ref,omitempty"`
	Status     string    `json:"status"`
}

var rememberCmd = &cobra.Command{
	Use:   "remember <content>",
	Short: "Store a memory for later recall",
	Long: `Store a memory with optional categorization and importance level.

Memories are notes with special tags for easy recall. They support:
- Categories: user-pref, project, decision, fact, todo
- Importance levels: 1-5 (default 3)
- TTL: Auto-expire after a duration
- Source tracking: Where the memory came from

Examples:
  noted remember "Always use snake_case for database columns"
  noted remember "Project uses PostgreSQL" --category project --importance 4
  noted remember "Temp note" --ttl 24h
  noted remember "Bug found" --source code-review --source-ref auth.go:142`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		content := args[0]
		title, _ := cmd.Flags().GetString("title")
		category, _ := cmd.Flags().GetString("category")
		importance, _ := cmd.Flags().GetInt("importance")
		ttlStr, _ := cmd.Flags().GetString("ttl")
		source, _ := cmd.Flags().GetString("source")
		sourceRef, _ := cmd.Flags().GetString("source-ref")
		asJSON, _ := cmd.Flags().GetBool("json")

		// Parse TTL if provided
		var ttl time.Duration
		if ttlStr != "" {
			var err error
			ttl, err = parseDuration(ttlStr)
			if err != nil {
				return fmt.Errorf("invalid TTL: %w", err)
			}
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
		mem, err := memory.Remember(ctx, database, syncer, memory.RememberInput{
			Content:    content,
			Title:      title,
			Category:   category,
			Importance: importance,
			TTL:        ttl,
			Source:     source,
			SourceRef:  sourceRef,
		})
		if err != nil {
			return err
		}

		if asJSON {
			result := rememberResult{
				ID:         mem.ID,
				Title:      mem.Title,
				Category:   mem.Category,
				Importance: mem.Importance,
				Status:     "remembered",
			}
			if !mem.ExpiresAt.IsZero() {
				result.ExpiresAt = mem.ExpiresAt.Format(time.RFC3339)
			}
			if mem.Source != "" {
				result.Source = mem.Source
			}
			if mem.SourceRef != "" {
				result.SourceRef = mem.SourceRef
			}
			return outputJSON(result)
		}

		fmt.Printf("Remembered #%d: %s\n", mem.ID, mem.Title)
		fmt.Printf("  Category:   %s\n", mem.Category)
		fmt.Printf("  Importance: %d\n", mem.Importance)
		if !mem.ExpiresAt.IsZero() {
			fmt.Printf("  Expires:    %s\n", mem.ExpiresAt.Format("2006-01-02 15:04"))
		}
		if mem.Source != "" {
			fmt.Printf("  Source:     %s\n", mem.Source)
			if mem.SourceRef != "" {
				fmt.Printf("  Reference:  %s\n", mem.SourceRef)
			}
		}
		return nil
	},
}

// parseDuration parses a duration string with support for days (e.g., "7d", "24h")
func parseDuration(s string) (time.Duration, error) {
	// Check for day suffix
	if len(s) > 0 && s[len(s)-1] == 'd' {
		days := s[:len(s)-1]
		var n int
		if _, err := fmt.Sscanf(days, "%d", &n); err != nil {
			return 0, fmt.Errorf("invalid day format: %s", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func init() {
	rootCmd.AddCommand(rememberCmd)

	rememberCmd.Flags().StringP("title", "t", "", "Short title for the memory")
	rememberCmd.Flags().StringP("category", "c", "fact", "Category: user-pref, project, decision, fact, todo")
	rememberCmd.Flags().IntP("importance", "i", 3, "Importance level 1-5")
	rememberCmd.Flags().String("ttl", "", "Time-to-live duration (e.g., '24h', '7d')")
	rememberCmd.Flags().String("source", "", "Source identifier (e.g., 'code-review', 'manual')")
	rememberCmd.Flags().String("source-ref", "", "Source reference (e.g., 'main.go:50')")
	rememberCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
