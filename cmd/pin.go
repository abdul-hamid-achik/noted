/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var pinCmd = &cobra.Command{
	Use:   "pin <id>",
	Short: "Pin a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid note ID: %s", args[0])
		}

		ctx := context.Background()

		// Verify note exists
		note, err := database.GetNote(ctx, id)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("note #%d not found", id)
			}
			return err
		}

		if err := database.PinNote(ctx, id); err != nil {
			return fmt.Errorf("failed to pin note: %w", err)
		}

		if asJSON {
			return outputJSON(map[string]any{
				"id":     id,
				"title":  note.Title,
				"pinned": true,
			})
		}

		fmt.Printf("Pinned note #%d: %s\n", id, note.Title)
		return nil
	},
}

var unpinCmd = &cobra.Command{
	Use:   "unpin <id>",
	Short: "Unpin a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid note ID: %s", args[0])
		}

		ctx := context.Background()

		// Verify note exists
		note, err := database.GetNote(ctx, id)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("note #%d not found", id)
			}
			return err
		}

		if err := database.UnpinNote(ctx, id); err != nil {
			return fmt.Errorf("failed to unpin note: %w", err)
		}

		if asJSON {
			return outputJSON(map[string]any{
				"id":     id,
				"title":  note.Title,
				"pinned": false,
			})
		}

		fmt.Printf("Unpinned note #%d: %s\n", id, note.Title)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pinCmd)
	rootCmd.AddCommand(unpinCmd)

	pinCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	unpinCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
