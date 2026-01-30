/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/abdul-hamid-achik/noted/internal/config"
	notedmcp "github.com/abdul-hamid-achik/noted/internal/mcp"
	"github.com/abdul-hamid-achik/noted/internal/veclite"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for AI agent integration",
	Long: `Start an MCP (Model Context Protocol) server that exposes noted
functionality to AI agents like Claude.

The server runs on stdio and provides tools for:
  - Creating, reading, updating, and deleting notes
  - Searching notes by text or semantic similarity
  - Managing tags
  - Memory tools for agents (remember, recall, forget)

Environment variables:
  NOTED_VECLITE_PATH     Path to veclite database for semantic search (optional)
  NOTED_EMBEDDING_MODEL  Embedding model name (default: nomic-embed-text)
  OLLAMA_HOST            Ollama server URL (default: http://localhost:11434)

Example usage with Claude Code:
  claude mcp add noted -- noted mcp`,
	RunE: runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

func runMCP(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize database (bypass PersistentPreRunE since we need custom handling)
	if database == nil {
		return fmt.Errorf("database not initialized")
	}

	// Try to initialize veclite syncer (optional)
	var syncer notedmcp.Syncer
	if cfg.VeclitePath != "" {
		s, err := veclite.NewSyncer(cfg.VeclitePath, cfg.EmbeddingModel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: veclite initialization failed: %v\n", err)
			fmt.Fprintf(os.Stderr, "Semantic search will be disabled\n")
		} else {
			syncer = s
			defer s.Close()
		}
	}

	// Create MCP server
	server := notedmcp.NewServer(database, syncer)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Run MCP server
	return server.Run(ctx)
}
