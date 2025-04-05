package cmd

import (
	"os"

	"github.com/sid-technologies/centurion/lib/errors"
	"github.com/spf13/cobra"
)

func CompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate completion script",
		Long: `To load completions:

Bash:
  source <(centurion completion bash)

Zsh:
  centurion completion zsh > "${fpath[1]}/_mycli"

Fish:
  centurion completion fish | source

PowerShell:
  centurion completion powershell | Out-String | Invoke-Expression
`,
		PreRunE: func(_ *cobra.Command, args []string) error {
			validShells := map[string]bool{
				"bash":       true,
				"zsh":        true,
				"fish":       true,
				"powershell": true,
			}
			if !validShells[args[0]] {
				return errors.New("unsupported shell type %q. Supported: bash, zsh, fish, powershell", args[0])
			}

			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			shell := args[0]
			switch shell {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return errors.New("unsupported shell type: %s", shell)
			}
		},
	}

	return cmd
}

//nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(CompletionCmd())
}
