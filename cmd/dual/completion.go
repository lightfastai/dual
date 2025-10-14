package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

  $ source <(dual completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ dual completion bash > /etc/bash_completion.d/dual
  # macOS:
  $ dual completion bash > $(brew --prefix)/etc/bash_completion.d/dual

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ dual completion zsh > "${fpath[1]}/_dual"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ dual completion fish | source

  # To load completions for each session, execute once:
  $ dual completion fish > ~/.config/fish/completions/dual.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		default:
			return fmt.Errorf("unsupported shell type %q", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

// serviceCompletion provides dynamic completion for service names from dual.config.yml
func serviceCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Load config to get service names
	cfg, _, err := config.LoadConfig()
	if err != nil {
		// If config can't be loaded, return empty list with no file completion
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	// Extract service names
	services := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		services = append(services, name)
	}

	// Sort for consistent ordering
	sort.Strings(services)

	return services, cobra.ShellCompDirectiveNoFileComp
}

// contextCompletion provides dynamic completion for context names from the registry
func contextCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Try to get project root
	cfg, projectRoot, err := config.LoadConfig()
	if err != nil {
		// If config can't be loaded, return empty list
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	// Get project identifier
	projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	// Load registry
	reg, err := registry.LoadRegistry()
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}
	defer reg.Close()

	// Get contexts for this project
	contexts, err := reg.ListContexts(projectIdentifier)
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	// Extract context names
	contextNames := make([]string, 0, len(contexts))
	for name := range contexts {
		contextNames = append(contextNames, name)
	}

	// Sort for consistent ordering
	sort.Strings(contextNames)

	// Prevent file completion
	_ = cfg // Use cfg to avoid unused variable warning

	return contextNames, cobra.ShellCompDirectiveNoFileComp
}

// serviceNameCompletion provides completion for service names (for service remove command)
// This is the same as serviceCompletion but with a different signature for ValidArgsFunction
func serviceNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return serviceCompletion(cmd, args, toComplete)
}
