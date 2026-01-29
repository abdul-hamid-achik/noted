/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List all tags",
	RunE: func(cmd *cobra.Command, args []string) error {
		showCount, _ := cmd.Flags().GetBool("count")
		deleteUnused, _ := cmd.Flags().GetBool("delete-unused")

		ctx := context.Background()

		if deleteUnused {
			count, err := database.DeleteUnusedTags(ctx)
			if err != nil {
				return err
			}
			fmt.Printf("Deleted %d unused tag(s).\n", count)
			return nil
		}

		if showCount {
			tags, err := database.GetTagsWithCount(ctx)
			if err != nil {
				return err
			}

			if len(tags) == 0 {
				fmt.Println("No tags found.")
				return nil
			}

			for _, tag := range tags {
				fmt.Printf("%s (%d)\n", tag.Name, tag.NoteCount)
			}
		} else {
			tags, err := database.ListTags(ctx)
			if err != nil {
				return err
			}

			if len(tags) == 0 {
				fmt.Println("No tags found.")
				return nil
			}

			for _, tag := range tags {
				fmt.Println(tag.Name)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tagsCmd)

	tagsCmd.Flags().BoolP("count", "c", false, "Show note count per tag")
	tagsCmd.Flags().BoolP("delete-unused", "d", false, "Delete orphan tags")
}
