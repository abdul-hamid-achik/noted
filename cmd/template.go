/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/spf13/cobra"
)

type templateItem struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func formatTemplateItem(t db.Template) templateItem {
	item := templateItem{
		ID:      t.ID,
		Name:    t.Name,
		Content: t.Content,
	}
	if t.CreatedAt.Valid {
		item.CreatedAt = t.CreatedAt.Time.Format(time.RFC3339)
	}
	if t.UpdatedAt.Valid {
		item.UpdatedAt = t.UpdatedAt.Time.Format(time.RFC3339)
	}
	return item
}

func interpolateTemplate(content, title string) string {
	now := time.Now()
	r := strings.NewReplacer(
		"{{date}}", now.Format("2006-01-02"),
		"{{time}}", now.Format("15:04"),
		"{{datetime}}", now.Format("2006-01-02 15:04"),
		"{{title}}", title,
	)
	return r.Replace(content)
}

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage note templates",
	Long:  "Create, list, show, edit, and delete note templates.",
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")

		ctx := context.Background()
		templates, err := database.ListTemplates(ctx)
		if err != nil {
			return err
		}

		if asJSON {
			items := make([]templateItem, len(templates))
			for i, t := range templates {
				items[i] = formatTemplateItem(t)
			}
			return outputJSON(items)
		}

		if len(templates) == 0 {
			fmt.Println("No templates found.")
			return nil
		}

		for _, t := range templates {
			fmt.Printf("#%-4d %s\n", t.ID, t.Name)
		}

		return nil
	},
}

var templateCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new template",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		content, _ := cmd.Flags().GetString("content")
		asJSON, _ := cmd.Flags().GetBool("json")

		if content == "" {
			var err error
			content, err = openEditor()
			if err != nil {
				return err
			}
		}

		ctx := context.Background()
		tmpl, err := database.CreateTemplate(ctx, db.CreateTemplateParams{
			Name:    name,
			Content: content,
		})
		if err != nil {
			return err
		}

		if asJSON {
			return outputJSON(formatTemplateItem(tmpl))
		}

		fmt.Printf("Created template #%d: %s\n", tmpl.ID, tmpl.Name)
		return nil
	},
}

var templateShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show a template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")

		ctx := context.Background()
		tmpl, err := database.GetTemplateByName(ctx, args[0])
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("template %q not found", args[0])
			}
			return err
		}

		if asJSON {
			return outputJSON(formatTemplateItem(tmpl))
		}

		fmt.Printf("Template: %s (#%d)\n\n%s\n", tmpl.Name, tmpl.ID, tmpl.Content)
		return nil
	},
}

var templateDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")
		force, _ := cmd.Flags().GetBool("force")

		ctx := context.Background()
		tmpl, err := database.GetTemplateByName(ctx, args[0])
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("template %q not found", args[0])
			}
			return err
		}

		if !force {
			fmt.Printf("Delete template %q (#%d)? [y/N]: ", tmpl.Name, tmpl.ID)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		if err := database.DeleteTemplateByName(ctx, args[0]); err != nil {
			return fmt.Errorf("failed to delete template: %w", err)
		}

		if asJSON {
			return outputJSON(map[string]any{
				"deleted_id":   tmpl.ID,
				"deleted_name": tmpl.Name,
			})
		}

		fmt.Printf("Deleted template #%d: %s\n", tmpl.ID, tmpl.Name)
		return nil
	},
}

var templateEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit a template in your editor",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")

		ctx := context.Background()
		tmpl, err := database.GetTemplateByName(ctx, args[0])
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("template %q not found", args[0])
			}
			return err
		}

		newContent, err := openEditorWithContent(tmpl.Content)
		if err != nil {
			return err
		}

		updated, err := database.UpdateTemplate(ctx, db.UpdateTemplateParams{
			Content: newContent,
			ID:      tmpl.ID,
		})
		if err != nil {
			return err
		}

		if asJSON {
			return outputJSON(formatTemplateItem(updated))
		}

		fmt.Printf("Updated template #%d: %s\n", updated.ID, updated.Name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateCreateCmd)
	templateCmd.AddCommand(templateShowCmd)
	templateCmd.AddCommand(templateDeleteCmd)
	templateCmd.AddCommand(templateEditCmd)

	templateListCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	templateCreateCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	templateCreateCmd.Flags().StringP("name", "n", "", "Template name (required)")
	templateCreateCmd.Flags().StringP("content", "c", "", "Template content")
	_ = templateCreateCmd.MarkFlagRequired("name")
	templateShowCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	templateDeleteCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	templateDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation")
	templateEditCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
