/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id> [id...]",
	Short: "Delete notes",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		ids := make([]int64, 0, len(args))
		for _, arg := range args {
			id, err := strconv.ParseInt(arg, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid note ID: %s", arg)
			}
			ids = append(ids, id)
		}

		if !force {
			fmt.Printf("Delete %d note(s)? [y/N]: ", len(ids))
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

		ctx := context.Background()
		deleted := 0
		for _, id := range ids {
			_, err := database.GetNote(ctx, id)
			if err != nil {
				if err == sql.ErrNoRows {
					fmt.Fprintf(os.Stderr, "note #%d not found\n", id)
					continue
				}
				return err
			}

			if err := database.DeleteNote(ctx, id); err != nil {
				return fmt.Errorf("failed to delete note #%d: %w", id, err)
			}
			fmt.Printf("Deleted note #%d\n", id)
			deleted++
		}

		if deleted > 0 {
			fmt.Printf("\n%d note(s) deleted.\n", deleted)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation")
}
