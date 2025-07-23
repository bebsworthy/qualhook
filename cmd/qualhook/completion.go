// Package main provides shell completion commands for qualhook
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for qualhook.

Shell completions enable tab-completion for commands, flags, and arguments,
making the CLI more convenient to use.

To load completions:

BASH:
  # Linux:
  $ qualhook completion bash > /etc/bash_completion.d/qualhook

  # macOS:
  $ qualhook completion bash > $(brew --prefix)/etc/bash_completion.d/qualhook

  # Per-session:
  $ source <(qualhook completion bash)

ZSH:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ qualhook completion zsh > "${fpath[1]}/_qualhook"

  # Or for oh-my-zsh users:
  $ qualhook completion zsh > ~/.oh-my-zsh/completions/_qualhook

  # You will need to start a new shell for this setup to take effect.

FISH:
  $ qualhook completion fish | source

  # To load completions for each session, execute once:
  $ qualhook completion fish > ~/.config/fish/completions/qualhook.fish

POWERSHELL:
  PS> qualhook completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> qualhook completion powershell > qualhook.ps1
  # and source this file from your PowerShell profile.`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			if err := cmd.Root().GenBashCompletion(os.Stdout); err != nil {
				return fmt.Errorf("failed to generate bash completion: %w", err)
			}
		case "zsh":
			if err := cmd.Root().GenZshCompletion(os.Stdout); err != nil {
				return fmt.Errorf("failed to generate zsh completion: %w", err)
			}
		case "fish":
			if err := cmd.Root().GenFishCompletion(os.Stdout, true); err != nil {
				return fmt.Errorf("failed to generate fish completion: %w", err)
			}
		case "powershell":
			if err := cmd.Root().GenPowerShellCompletion(os.Stdout); err != nil {
				return fmt.Errorf("failed to generate powershell completion: %w", err)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}