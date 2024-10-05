package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

$ source <(qrcp completion bash)

# To load completions for each session, execute once:
Linux:
  $ qrcp completion bash > /etc/bash_completion.d/qrcp
MacOS:
  $ qrcp completion bash > /usr/local/etc/bash_completion.d/qrcp

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ qrcp completion zsh > "${fpath[1]}/_qrcp"

# You will need to start a new shell for this setup to take effect.

Fish:

$ qrcp completion fish | source

# To load completions for each session, execute once:
$ qrcp completion fish > ~/.config/fish/completions/qrcp.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			if err := cmd.Root().GenBashCompletion(os.Stdout); err != nil {
				panic(err)
			}
		case "zsh":
			if err := cmd.Root().GenZshCompletion(os.Stdout); err != nil {
				panic(err)
			}
		case "fish":
			if err := cmd.Root().GenFishCompletion(os.Stdout, true); err != nil {
				panic(err)
			}
		case "powershell":
			if err := cmd.Root().GenPowerShellCompletion(os.Stdout); err != nil {
				panic(err)
			}
		}
	},
}
