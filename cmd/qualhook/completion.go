// Package main provides shell completion commands for qualhook
package main

import (
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
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			_ = cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			_ = cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			_ = cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			_ = cmd.Root().GenPowerShellCompletion(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}