/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for noted.

To load completions:

Bash:
  $ source <(noted completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ noted completion bash > /etc/bash_completion.d/noted
  # macOS:
  $ noted completion bash > $(brew --prefix)/etc/bash_completion.d/noted

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ noted completion zsh > "${fpath[1]}/_noted"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ noted completion fish | source

  # To load completions for each session, execute once:
  $ noted completion fish > ~/.config/fish/completions/noted.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletionV2(os.Stdout, true)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
