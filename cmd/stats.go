/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/abdul-hamid-achik/noted/internal/config"
	"github.com/spf13/cobra"
)

type statsResult struct {
	Notes  int64  `json:"notes"`
	Tags   int64  `json:"tags"`
	DBSize int64  `json:"db_size_bytes"`
	DBPath string `json:"db_path"`
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show knowledge base statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")

		ctx := context.Background()

		noteCount, err := database.CountNotes(ctx)
		if err != nil {
			return err
		}

		tagCount, err := database.CountTags(ctx)
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		var dbSize int64
		if info, err := os.Stat(cfg.DBPath); err == nil {
			dbSize = info.Size()
		}

		if asJSON {
			return outputJSON(statsResult{
				Notes:  noteCount,
				Tags:   tagCount,
				DBSize: dbSize,
				DBPath: cfg.DBPath,
			})
		}

		fmt.Printf("%-12s %d\n", "Notes:", noteCount)
		fmt.Printf("%-12s %d\n", "Tags:", tagCount)
		fmt.Printf("%-12s %s\n", "DB size:", formatBytes(dbSize))
		fmt.Printf("%-12s %s\n", "DB path:", cfg.DBPath)

		return nil
	},
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(statsCmd)

	statsCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
