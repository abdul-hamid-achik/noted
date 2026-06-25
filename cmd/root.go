/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"github.com/abdul-hamid-achik/noted/internal/config"
	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/tui"
	"github.com/abdul-hamid-achik/noted/internal/vault"
	"github.com/spf13/cobra"
)

var (
	database *db.Queries
	conn     *sql.DB
)

var rootCmd = &cobra.Command{
	Use:   "noted",
	Short: "A CLI knowledge base",
	Long:  "A simple CLI knowledge base application to store and manage your notes efficiently.",
	RunE:  runTUI,

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		dbPath, _ := cmd.Flags().GetString("db")
		if dbPath != "" {
			cfg.DBPath = dbPath
		}

		conn, err = db.Open(cfg.DBPath)
		if err != nil {
			return err
		}

		database = db.New(conn)

		return nil
	},

	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if conn != nil {
			_ = conn.Close()
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("db", "", "Path to database file")
	rootCmd.PersistentFlags().String("vault", "", "Path to the markdown vault directory (overrides $NOTED_VAULT)")
}

// vaultDir resolves the vault directory: --vault flag, else config (which honors $NOTED_VAULT).
func vaultDir(cmd *cobra.Command) string {
	if v, _ := cmd.Flags().GetString("vault"); v != "" {
		return v
	}
	if cfg, err := config.Load(); err == nil {
		return cfg.VaultPath
	}
	return ""
}

// openVault opens the markdown vault for write-through (best-effort; returns nil on failure).
func openVault(cmd *cobra.Command) *vault.Vault {
	v, err := vault.Open(vaultDir(cmd))
	if err != nil {
		return nil
	}
	return v
}

func outputJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runTUI(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Open the markdown vault for write-through (best-effort — TUI still works without it).
	vlt, _ := vault.Open(vaultDir(cmd))

	program, err := tui.New(ctx, conn, database, vlt)
	if err != nil {
		return fmt.Errorf("failed to initialize TUI: %w", err)
	}

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return nil
}
