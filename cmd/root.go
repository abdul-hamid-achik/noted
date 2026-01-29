/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>

*/
package cmd

import (
	"os"

	"database/sql"
	"github.com/spf13/cobra"
	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/config"
)

var (
	database *db.Queries
	conn *sql.DB
)

var rootCmd = &cobra.Command{
	Use:   "noted",
	Short: "A CLI knowledge base",
	Long: "A simple CLI knowledge base application to store and manage your notes efficiently.",

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
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
			conn.Close()
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
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}


