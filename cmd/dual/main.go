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
	Short: "Manage port assignments across development contexts",
	Long: `dual is a CLI tool that manages port assignments across different
development contexts (git branches, worktrees, or clones). It eliminates
port conflicts when working on multiple features simultaneously by
automatically detecting the context and service, then injecting the
appropriate PORT environment variable.`,
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, show help
		_ = cmd.Help()
	},
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
