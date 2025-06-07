package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// newCompletionCmd creates and returns the completion command
func newCompletionCmd() *cobra.Command {
	var completionCmd = &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script for specified shell",
		Long: `To load completions:

Bash:

$ source <(gh-octoscope completion bash)

# To load completions for each session, execute once:
Linux:
  $ gh-octoscope completion bash > /etc/bash_completion.d/gh-octoscope
MacOS:
  $ gh-octoscope completion bash > /usr/local/etc/bash_completion.d/gh-octoscope

Zsh:

$ source <(gh-octoscope completion zsh)

# To load completions for each session, execute once:
$ gh-octoscope completion zsh > "${fpath[1]}/_gh-octoscope"

Fish:

$ gh-octoscope completion fish | source

# To load completions for each session, execute once:
$ gh-octoscope completion fish > ~/.config/fish/completions/gh-octoscope.fish

PowerShell:

PS> gh-octoscope completion powershell | Out-String | Invoke-Expression

# To load completions for every new session, run:
PS> gh-octoscope completion powershell > gh-octoscope.ps1
# and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}
	return completionCmd
}
