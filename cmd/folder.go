/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/spf13/cobra"
)

type folderItem struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	ParentID *int64 `json:"parent_id,omitempty"`
}

var folderCmd = &cobra.Command{
	Use:   "folder",
	Short: "Manage folders",
	Long:  "Create, list, and delete folders for organizing notes.",
}

var folderListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all folders",
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")

		ctx := context.Background()
		folders, err := database.ListFolders(ctx)
		if err != nil {
			return err
		}

		if asJSON {
			items := make([]folderItem, len(folders))
			for i, f := range folders {
				items[i] = folderItem{ID: f.ID, Name: f.Name}
				if f.ParentID.Valid {
					pid := f.ParentID.Int64
					items[i].ParentID = &pid
				}
			}
			return outputJSON(items)
		}

		if len(folders) == 0 {
			fmt.Println("No folders found.")
			return nil
		}

		for _, f := range folders {
			if f.ParentID.Valid {
				fmt.Printf("#%-4d %-30s (parent: %d)\n", f.ID, f.Name, f.ParentID.Int64)
			} else {
				fmt.Printf("#%-4d %s\n", f.ID, f.Name)
			}
		}

		return nil
	},
}

var folderCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new folder",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")
		parentID, _ := cmd.Flags().GetInt64("parent")

		var parentVal sql.NullInt64
		if cmd.Flags().Changed("parent") {
			parentVal = sql.NullInt64{Int64: parentID, Valid: true}
		}

		ctx := context.Background()
		folder, err := database.CreateFolder(ctx, db.CreateFolderParams{
			Name:     args[0],
			ParentID: parentVal,
		})
		if err != nil {
			return err
		}

		if asJSON {
			item := folderItem{ID: folder.ID, Name: folder.Name}
			if folder.ParentID.Valid {
				pid := folder.ParentID.Int64
				item.ParentID = &pid
			}
			return outputJSON(item)
		}

		fmt.Printf("Created folder #%d: %s\n", folder.ID, folder.Name)
		return nil
	},
}

var folderDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a folder",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")
		force, _ := cmd.Flags().GetBool("force")

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid folder ID: %s", args[0])
		}

		ctx := context.Background()

		// Verify folder exists
		folder, err := database.GetFolder(ctx, id)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("folder #%d not found", id)
			}
			return err
		}

		if !force {
			fmt.Printf("Delete folder %q (#%d)? [y/N]: ", folder.Name, id)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		if err := database.DeleteFolder(ctx, id); err != nil {
			return fmt.Errorf("failed to delete folder: %w", err)
		}

		if asJSON {
			return outputJSON(map[string]any{
				"deleted_id":   id,
				"deleted_name": folder.Name,
			})
		}

		fmt.Printf("Deleted folder #%d: %s\n", id, folder.Name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(folderCmd)
	folderCmd.AddCommand(folderListCmd)
	folderCmd.AddCommand(folderCreateCmd)
	folderCmd.AddCommand(folderDeleteCmd)

	folderListCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	folderCreateCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	folderCreateCmd.Flags().Int64("parent", 0, "Parent folder ID")
	folderDeleteCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	folderDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation")
}
