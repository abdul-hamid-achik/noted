/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/abdul-hamid-achik/noted/internal/config"
	"github.com/abdul-hamid-achik/noted/internal/veclite"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync notes to veclite for semantic search",
	Long: `Sync notes to veclite for semantic search.

This command syncs unembedded notes to the veclite vector database,
enabling semantic search capabilities. Notes that have already been
synced will be skipped unless --force is used.

Environment variables:
  NOTED_VECLITE_PATH     Path to veclite database (required)
  NOTED_EMBEDDING_MODEL  Embedding model name (default: nomic-embed-text)
  OLLAMA_HOST            Ollama server URL (default: http://localhost:11434)

Example:
  noted sync           # Sync only unsynced notes
  noted sync --force   # Re-sync all notes`,
	RunE: runSync,
}

var syncForce bool

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolVarP(&syncForce, "force", "f", false, "Re-sync all notes even if already synced")
}

func runSync(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check veclite path is configured
	if cfg.VeclitePath == "" {
		return fmt.Errorf("NOTED_VECLITE_PATH environment variable is not set")
	}

	// Initialize database
	if database == nil {
		return fmt.Errorf("database not initialized")
	}

	// Create syncer
	syncer, err := veclite.NewSyncer(cfg.VeclitePath, cfg.EmbeddingModel)
	if err != nil {
		return fmt.Errorf("failed to initialize veclite: %w", err)
	}
	defer syncer.Close()

	ctx := context.Background()

	if syncForce {
		// Get all notes
		notes, err := database.GetAllNotes(ctx)
		if err != nil {
			return fmt.Errorf("failed to get notes: %w", err)
		}

		fmt.Printf("Syncing %d notes...\n", len(notes))

		synced := 0
		failed := 0
		for _, note := range notes {
			if err := syncer.SyncNote(note.ID, note.Title, note.Content); err != nil {
				fmt.Printf("  Failed to sync note #%d: %v\n", note.ID, err)
				failed++
				continue
			}
			_ = database.MarkEmbeddingSynced(ctx, note.ID)
			synced++
		}

		fmt.Printf("\nDone! Synced: %d, Failed: %d\n", synced, failed)
	} else {
		// Sync only unsynced notes
		synced, err := syncer.SyncAll(database)
		if err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		if synced == 0 {
			fmt.Println("All notes are already synced.")
		} else {
			fmt.Printf("Done! Synced %d notes.\n", synced)
		}
	}

	return nil
}
