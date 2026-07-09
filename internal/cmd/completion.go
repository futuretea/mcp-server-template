package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newCompletionCommand(streams IOStreams) *cobra.Command {
	command := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell|install]",
		Short: "generate shell completion script",
		Long: `To load completions:

  Bash:
    source <(mcp-server completion bash)

  Zsh:
    source <(mcp-server completion zsh)

  Fish:
    mcp-server completion fish | source

  PowerShell:
    mcp-server completion powershell | Out-String | Invoke-Expression`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell", "install"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "install":
				printCompletionInstall(streams.Out)
			case "bash":
				_ = cmd.Root().GenBashCompletion(streams.Out)
			case "zsh":
				_ = cmd.Root().GenZshCompletion(streams.Out)
			case "fish":
				_ = cmd.Root().GenFishCompletion(streams.Out, true)
			case "powershell":
				_ = cmd.Root().GenPowerShellCompletionWithDesc(streams.Out)
			}
		},
	}
	command.SetOut(streams.Out)
	command.SetErr(streams.ErrOut)
	return command
}

func printCompletionInstall(out io.Writer) {
	_, _ = fmt.Fprintln(out, `# Add the appropriate line to your shell profile:

# bash (~/.bashrc or ~/.bash_profile):
source <(mcp-server completion bash)

# zsh (~/.zshrc):
source <(mcp-server completion zsh)

# fish (~/.config/fish/config.fish):
mcp-server completion fish | source

# PowerShell ($PROFILE):
mcp-server completion powershell | Out-String | Invoke-Expression`)
}
