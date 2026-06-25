/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/abdul-hamid-achik/noted/internal/config"
	"github.com/abdul-hamid-achik/noted/internal/notesync"
	"github.com/abdul-hamid-achik/noted/internal/vault"
	"github.com/spf13/cobra"
)

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Manage the markdown vault (.md files)",
	Long: `Work with the markdown vault — plain .md files (with YAML frontmatter) that mirror your
notes so agents, editors, and git can read them directly.

  noted vault path            Print the vault directory
  noted vault export          Write every note out to the vault as a .md file
  noted vault import          Rebuild the SQLite index from the vault (--force to apply)`,
}

func resolveVaultPath(cmd *cobra.Command) (string, error) {
	if p, _ := cmd.Flags().GetString("path"); p != "" {
		return p, nil
	}
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	return cfg.VaultPath, nil
}

// vaultCmdVaultDir resolves the vault path for vault subcommands, preferring --path if present, then the
// persistent --vault/root flag, then config/env.
func vaultCmdVaultDir(cmd *cobra.Command) string {
	if p, _ := cmd.Flags().GetString("path"); p != "" {
		return p
	}
	return vaultDir(cmd)
}

var vaultPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the vault directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := resolveVaultPath(cmd)
		if err != nil {
			return err
		}
		if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
			return outputJSON(map[string]string{"path": p})
		}
		fmt.Println(p)
		return nil
	},
}

var vaultExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export all notes to the vault as markdown files",
	RunE: func(cmd *cobra.Command, args []string) error {
		vpath := vaultCmdVaultDir(cmd)
		if vpath == "" {
			return fmt.Errorf("no vault path: set --path, --vault, or $NOTED_VAULT")
		}
		vlt, err := vault.Open(vpath)
		if err != nil {
			return err
		}
		ctx := context.Background()
		notes, err := database.GetAllNotes(ctx)
		if err != nil {
			return err
		}

		seen := map[string]bool{} // keep filenames stable + collision-free
		count := 0
		for _, n := range notes {
			tags, _ := database.GetTagsForNote(ctx, n.ID)
			tnames := make([]string, len(tags))
			for i, t := range tags {
				tnames[i] = t.Name
			}
			base := vault.Slugify(n.Title)
			name := base
			for i := 2; seen[name]; i++ { // key uniqueness on the final filename, not the base slug
				name = fmt.Sprintf("%s-%d", base, i)
			}
			seen[name] = true

			vn := vault.Note{
				ID:      n.ID,
				Path:    filepath.Join(vpath, name+".md"),
				Title:   n.Title,
				Tags:    tnames,
				Pinned:  n.Pinned.Valid && n.Pinned.Bool,
				Content: n.Content,
			}
			if n.FolderID.Valid {
				vn.Folder = notesync.FolderPath(ctx, database, n.FolderID.Int64)
			}
			if n.CreatedAt.Valid {
				vn.Created = n.CreatedAt.Time
			}
			if n.UpdatedAt.Valid {
				vn.Updated = n.UpdatedAt.Time
			}
			if _, err := vlt.WriteRaw(vn); err != nil {
				return err
			}
			count++
		}

		// Also persist version history into the vault (.noted/versions/) so it survives a rebuild.
		versions, _ := notesync.PersistVersions(ctx, database, vlt)

		if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
			return outputJSON(map[string]any{"exported": count, "versions": versions, "path": vpath})
		}
		fmt.Printf("Exported %d notes (%d version snapshots) to %s\n", count, versions, vpath)
		return nil
	},
}

var vaultImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Rebuild the SQLite index from the vault's markdown files",
	Long: `Rebuild the SQLite index from the vault. The vault is treated as the source of truth: the
notes/tags/links/folders index is REPLACED with what's in the vault, preserving each note's id from
its frontmatter. Version history (.noted/versions/) is restored too. Runs as a preview unless --force.

Agent memories (noted remember) live only in the index, not the vault. An in-place rebuild preserves
them; rebuilding a brand-new, empty database from only a vault won't have them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		vpath := vaultCmdVaultDir(cmd)
		if vpath == "" {
			return fmt.Errorf("no vault path: set --path, --vault, or $NOTED_VAULT")
		}
		vlt, err := vault.Open(vpath)
		if err != nil {
			return err
		}
		notes, err := vlt.List()
		if err != nil {
			return err
		}
		asJSON, _ := cmd.Flags().GetBool("json")
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			if asJSON {
				return outputJSON(map[string]any{"would_import": len(notes), "path": vpath, "dry_run": true})
			}
			fmt.Printf("Would rebuild the index from %d notes in %s.\n", len(notes), vpath)
			fmt.Println("This REPLACES the current notes/tags/links index. Re-run with --force to apply.")
			return nil
		}

		ctx := context.Background()
		stats, err := notesync.Rebuild(ctx, conn, vlt)
		if err != nil {
			return err
		}

		if asJSON {
			return outputJSON(map[string]any{"imported": stats.Notes, "links": stats.Links, "remapped_ids": stats.RemappedIDs, "restored_versions": stats.RestoredVersions, "path": vpath})
		}
		summary := fmt.Sprintf("Rebuilt index from %s: %d notes, %d links.", vpath, stats.Notes, stats.Links)
		if stats.RestoredVersions > 0 {
			summary += fmt.Sprintf(" Restored %d version snapshot(s).", stats.RestoredVersions)
		}
		if stats.PreservedMemories > 0 {
			unit := "memories"
			if stats.PreservedMemories == 1 {
				unit = "memory"
			}
			summary += fmt.Sprintf(" Preserved %d %s.", stats.PreservedMemories, unit)
		}
		if stats.RemappedIDs > 0 {
			summary += fmt.Sprintf(" (%d duplicate id(s) remapped)", stats.RemappedIDs)
		}
		fmt.Println(summary)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(vaultCmd)
	vaultCmd.AddCommand(vaultPathCmd, vaultExportCmd, vaultImportCmd)

	for _, c := range []*cobra.Command{vaultPathCmd, vaultExportCmd, vaultImportCmd} {
		c.Flags().String("path", "", "Vault directory (overrides config / $NOTED_VAULT)")
		c.Flags().BoolP("json", "j", false, "Output as JSON")
	}
	vaultImportCmd.Flags().BoolP("force", "f", false, "Apply the rebuild (default is a dry-run preview)")
}
