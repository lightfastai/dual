package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// CLI entry point for the dual tool

var (
	// Version information - will be set via ldflags during build
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "dual",
	Short: "Manage worktree lifecycle with environment remapping",
	Long: `dual is a CLI tool that manages git worktree lifecycle (create, delete)
with environment remapping via hooks. It enables flexible development workflows
across multiple branches and worktrees, allowing users to implement custom
environment management logic through hooks.`,
	Version: version,
}

func init() {
	// Custom version template that includes commit and build date
	rootCmd.SetVersionTemplate(`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}
Commit: {{.Annotations.commit}}
Built: {{.Annotations.date}}
`)

	// Set annotations for version info
	if rootCmd.Annotations == nil {
		rootCmd.Annotations = make(map[string]string)
	}
	rootCmd.Annotations["commit"] = commit
	rootCmd.Annotations["date"] = date

	// Add version flag (cobra adds this automatically, but we ensure it's there)
	rootCmd.Flags().BoolP("version", "v", false, "version for dual")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
